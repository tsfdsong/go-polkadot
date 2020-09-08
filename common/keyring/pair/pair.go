package pair

import (
	"encoding/json"
	"errors"

	"github.com/tsfdsong/go-polkadot/common/crypto"
	"github.com/tsfdsong/go-polkadot/common/keyring/address"
	ktypes "github.com/tsfdsong/go-polkadot/common/keyring/types"
	"github.com/tsfdsong/go-polkadot/common/u8util"
	"github.com/tsfdsong/go-polkadot/logger"
)

// NewPair ...
func NewPair(naclPub [32]byte, naclPriv [64]byte, meta ktypes.Meta, defaultEncoded []byte) (*Pair, error) {
	state := &State{
		Meta:      meta,
		PublicKey: naclPub,
	}

	return &Pair{
		State:          state,
		defaultEncoded: defaultEncoded,
		secretKey:      naclPriv,
	}, nil
}

// NewPairFromJSON ...
func NewPairFromJSON(data []byte, password *string) (*Pair, error) {
	tmp := forJSON{}
	if err := json.Unmarshal(data, &tmp); err != nil {
		return nil, err
	}

	pubBytes, err := address.Decode(tmp.Address, nil)
	if err != nil {
		return nil, err
	}
	var (
		pub  [32]byte
		priv [64]byte
	)
	copy(pub[:], pubBytes)

	privBytes := u8util.FromHex(tmp.Encoded)
	pub2, priv2, err := Decode(password, privBytes)
	if err != nil {
		return nil, err
	}

	if len(pub2) != len(pub) {
		return nil, errors.New("public keys do not match")
	}
	for idx := range pub2 {
		if pub2[idx] != pub[idx] {
			return nil, errors.New("public keys do not match")
		}
	}
	copy(priv[:], priv2[:])

	// TODO: nil defaultEncoded?
	return NewPair(pub, priv, tmp.Meta, nil)
}

// Address ...
func (p *Pair) Address() (string, error) {
	if p.State == nil {
		return "", errors.New("nil state")
	}

	return address.Encode(p.State.PublicKey[:], nil)
}

// DecodePkcs8 ...
func (p *Pair) DecodePkcs8(passphrase *string, encoded []byte) error {
	tmp := p.defaultEncoded
	if encoded != nil && len(encoded) > 0 {
		tmp = encoded
	}

	pub, priv, err := Decode(passphrase, tmp)
	if err != nil {
		return err
	}

	p.State.PublicKey = pub
	p.secretKey = priv

	return nil
}

// EncodePkcs8 ...
func (p *Pair) EncodePkcs8(passphrase *string) ([]byte, error) {
	return Encode(p.secretKey, passphrase)
}

// GetMeta ...
func (p *Pair) GetMeta() (ktypes.Meta, error) {
	if p.State == nil {
		return nil, errors.New("nil state")
	}

	return p.State.Meta, nil
}

// IsLocked ...
func (p *Pair) IsLocked() bool {
	// note: string comparison is slow, better method? reflect.DeepEqual will probably also be slow...
	blankSecret := [64]byte{}
	return string(p.secretKey[:]) == string(blankSecret[:])
}

// Lock ...
func (p *Pair) Lock() error {
	p.secretKey = [64]byte{}
	return nil
}

// PublicKey ...
func (p *Pair) PublicKey() ([32]byte, error) {
	if p.State == nil {
		return [32]byte{}, errors.New("state is nil")
	}

	return p.State.PublicKey, nil
}

// SetMeta ...
func (p *Pair) SetMeta(meta ktypes.Meta) error {
	if p.State == nil {
		return errors.New("state is nil")
	}

	p.State.Meta = meta
	return nil
}

// Sign ...
func (p *Pair) Sign(message []byte) ([]byte, error) {
	return crypto.NaclSign(p.secretKey, message)
}

// ToJSON ...
func (p *Pair) ToJSON(passphrase *string) ([]byte, error) {
	if p.State == nil {
		return nil, errors.New("nil state")
	}

	var isEncrypted bool
	if passphrase != nil && *passphrase != "" {
		isEncrypted = true
	}

	encoded, err := Encode(p.secretKey, passphrase)
	if err != nil {
		logger.Errorf("err encoding secretkey\n%v", err)
		return nil, err
	}

	addr, err := address.Encode(p.State.PublicKey[:], nil)
	if err != nil {
		logger.Errorf("err encoding public key\n%v", err)
		return nil, err
	}

	typ := None
	if isEncrypted {
		typ = XSalsa20_Poly1305
	}
	enc := encoding{
		Content: PKCS8,
		Type:    typ,
		Version: "0",
	}
	tmp := forJSON{
		Address:  addr,
		Encoded:  u8util.ToHex(encoded, -1, false),
		Encoding: enc,
		Meta:     p.State.Meta,
	}
	return json.Marshal(tmp)
}

// Verify ...
func (p *Pair) Verify(message, signature []byte) (bool, error) {
	if message == nil || len(message) == 0 {
		return false, errors.New("canno verify a nil message")
	}
	if signature == nil || len(signature) == 0 {
		return false, errors.New("cannot verify a nil signature")
	}
	if p.State == nil {
		return false, errors.New("nil state")
	}

	return crypto.NaclVerify(message, signature, p.State.PublicKey), nil
}
