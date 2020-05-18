// Package fake provides fake implementations for interfaces commonly used in
// the repository.
// The implementations offer configuration to return errors when it is needed by
// the unit test and it is also possible to record the call of functions of an
// object in some cases.
package fake

import (
	"context"
	"encoding/binary"
	"fmt"
	"hash"
	"io"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes/any"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/golang/protobuf/ptypes/wrappers"
	"go.dedis.ch/fabric/consensus/viewchange"
	"go.dedis.ch/fabric/crypto"
	"go.dedis.ch/fabric/encoding"
	"go.dedis.ch/fabric/mino"
	"golang.org/x/xerrors"
)

// Call is a tool to keep track of a function calls.
type Call struct {
	calls [][]interface{}
}

// Get returns the nth call ith parameter.
func (c *Call) Get(n, i int) interface{} {
	if c == nil {
		return nil
	}

	return c.calls[n][i]
}

// Len returns the number of calls.
func (c *Call) Len() int {
	if c == nil {
		return 0
	}

	return len(c.calls)
}

// Add adds a call to the list.
func (c *Call) Add(args ...interface{}) {
	if c == nil {
		return
	}

	c.calls = append(c.calls, args)
}

// Address is a fake implementation of mino.Address
type Address struct {
	mino.Address
	index int
	err   error
}

// NewAddress returns a fake address with the given index.
func NewAddress(index int) Address {
	return Address{index: index}
}

// NewBadAddress returns a fake address that returns an error when appropriate.
func NewBadAddress() Address {
	return Address{err: xerrors.New("fake error")}
}

// Equal implements mino.Address.
func (a Address) Equal(o mino.Address) bool {
	other, ok := o.(Address)
	return ok && other.index == a.index
}

// MarshalText implements encoding.TextMarshaler.
func (a Address) MarshalText() ([]byte, error) {
	buffer := make([]byte, 4)
	binary.LittleEndian.PutUint32(buffer, uint32(a.index))
	return buffer, a.err
}

func (a Address) String() string {
	return fmt.Sprintf("fake.Address[%d]", a.index)
}

// AddressIterator is a fake implementation of the mino.AddressIterator
// interface.
type AddressIterator struct {
	mino.AddressIterator
	addrs []mino.Address
	index int
}

// NewAddressIterator returns a new address iterator
func NewAddressIterator(addrs []mino.Address) AddressIterator {
	return AddressIterator{
		addrs: addrs,
	}
}

// Seek implements mino.AddressIterator.
func (i *AddressIterator) Seek(index int) {
	i.index = index
}

// HasNext implements mino.AddressIterator.
func (i *AddressIterator) HasNext() bool {
	return i.index < len(i.addrs)
}

// GetNext implements mino.AddressIterator.
func (i *AddressIterator) GetNext() mino.Address {
	if i.HasNext() {
		res := i.addrs[i.index]
		i.index++
		return res
	}
	return nil
}

// PublicKeyIterator is a fake implementation of crypto.PublicKeyIterator.
type PublicKeyIterator struct {
	signers []crypto.AggregateSigner
	index   int
}

// Seek implements crypto.PublicKeyIterator.
func (i *PublicKeyIterator) Seek(index int) {
	i.index = index
}

// HasNext implements crypto.PublicKeyIterator.
func (i *PublicKeyIterator) HasNext() bool {
	return i.index < len(i.signers)
}

// GetNext implements crypto.PublicKeyIterator.
func (i *PublicKeyIterator) GetNext() crypto.PublicKey {
	if i.HasNext() {
		res := i.signers[i.index]
		i.index++
		return res.GetPublicKey()
	}
	return nil
}

// CollectiveAuthority is a fake implementation of the cosi.CollectiveAuthority
// interface.
type CollectiveAuthority struct {
	encoding.Packable
	crypto.CollectiveAuthority
	addrs   []mino.Address
	signers []crypto.AggregateSigner

	Call *Call
}

// GenSigner is a function to generate a signer.
type GenSigner func() crypto.AggregateSigner

// NewAuthority returns a new collective authority of n members with new signers
// generated by g.
func NewAuthority(n int, g GenSigner) CollectiveAuthority {
	return NewAuthorityWithBase(0, n, g)
}

// NewAuthorityWithBase returns a new fake collective authority of size n with
// a given starting base index.
func NewAuthorityWithBase(base int, n int, g GenSigner) CollectiveAuthority {
	signers := make([]crypto.AggregateSigner, n)
	for i := range signers {
		signers[i] = g()
	}

	addrs := make([]mino.Address, n)
	for i := range addrs {
		addrs[i] = Address{index: i + base}
	}

	return CollectiveAuthority{
		signers: signers,
		addrs:   addrs,
	}
}

// NewAuthorityFromMino returns a new fake collective authority using
// the addresses of the Mino instances.
func NewAuthorityFromMino(g func() crypto.AggregateSigner, instances ...mino.Mino) CollectiveAuthority {
	signers := make([]crypto.AggregateSigner, len(instances))
	for i := range signers {
		signers[i] = g()
	}

	addrs := make([]mino.Address, len(instances))
	for i, instance := range instances {
		addrs[i] = instance.GetAddress()
	}

	return CollectiveAuthority{
		signers: signers,
		addrs:   addrs,
	}
}

// GetAddress returns the address at the provided index.
func (ca CollectiveAuthority) GetAddress(index int) mino.Address {
	return ca.addrs[index]
}

// GetSigner returns the signer at the provided index.
func (ca CollectiveAuthority) GetSigner(index int) crypto.AggregateSigner {
	return ca.signers[index]
}

// GetPublicKey implements cosi.CollectiveAuthority.
func (ca CollectiveAuthority) GetPublicKey(addr mino.Address) (crypto.PublicKey, int) {
	for i, address := range ca.addrs {
		if address.Equal(addr) {
			return ca.signers[i].GetPublicKey(), i
		}
	}
	return nil, -1
}

// Take implements mino.Players.
func (ca CollectiveAuthority) Take(updaters ...mino.FilterUpdater) mino.Players {
	filter := mino.ApplyFilters(updaters)
	newCA := CollectiveAuthority{
		Call:    ca.Call,
		addrs:   make([]mino.Address, len(filter.Indices)),
		signers: make([]crypto.AggregateSigner, len(filter.Indices)),
	}
	for i, k := range filter.Indices {
		newCA.addrs[i] = ca.addrs[k]
		newCA.signers[i] = ca.signers[k]
	}
	return newCA
}

type signerWrapper struct {
	crypto.AggregateSigner
	pubkey crypto.PublicKey
}

func (s signerWrapper) GetPublicKey() crypto.PublicKey {
	return s.pubkey
}

// Apply implements viewchange.EvolvableAuthority.
func (ca CollectiveAuthority) Apply(cs viewchange.ChangeSet) viewchange.EvolvableAuthority {
	if ca.Call != nil {
		ca.Call.Add("apply", cs)
	}

	newAuthority := CollectiveAuthority{
		Call:    ca.Call,
		addrs:   make([]mino.Address, len(ca.addrs)),
		signers: make([]crypto.AggregateSigner, len(ca.signers)),
	}
	for i := range ca.addrs {
		newAuthority.addrs[i] = ca.addrs[i]
		newAuthority.signers[i] = ca.signers[i]
	}

	for _, player := range cs.Add {
		newAuthority.addrs = append(newAuthority.addrs, player.Address)
		newAuthority.signers = append(newAuthority.signers, signerWrapper{
			pubkey: player.PublicKey,
		})
	}

	for _, i := range cs.Remove {
		newAuthority.addrs = append(newAuthority.addrs[:i], newAuthority.addrs[i+1:]...)
		newAuthority.signers = append(newAuthority.signers[:i], newAuthority.signers[i+1:]...)
	}

	return newAuthority
}

// Len implements mino.Players.
func (ca CollectiveAuthority) Len() int {
	return len(ca.signers)
}

// AddressIterator implements mino.Players.
func (ca CollectiveAuthority) AddressIterator() mino.AddressIterator {
	return &AddressIterator{addrs: ca.addrs}
}

// PublicKeyIterator implements cosi.CollectiveAuthority.
func (ca CollectiveAuthority) PublicKeyIterator() crypto.PublicKeyIterator {
	return &PublicKeyIterator{signers: ca.signers}
}

// PublicKeyFactory is a fake implementation of a public key factory.
type PublicKeyFactory struct {
	crypto.PublicKeyFactory
	pubkey PublicKey
	err    error
}

// NewPublicKeyFactory returns a fake public key factory that returns the given
// public key.
func NewPublicKeyFactory(pubkey PublicKey) PublicKeyFactory {
	return PublicKeyFactory{
		pubkey: pubkey,
	}
}

// NewBadPublicKeyFactory returns a fake public key factory that returns an
// error when appropriate.
func NewBadPublicKeyFactory() PublicKeyFactory {
	return PublicKeyFactory{err: xerrors.New("fake error")}
}

// FromProto implements crypto.PublicKeyFactory.
func (f PublicKeyFactory) FromProto(proto.Message) (crypto.PublicKey, error) {
	return f.pubkey, f.err
}

// SignatureByte is the byte returned when marshaling a fake signature.
const SignatureByte = 0xfe

// Signature is a fake implementation of the signature.
type Signature struct {
	crypto.Signature
	err error
}

// NewBadSignature returns a signature that will return error when appropriate.
func NewBadSignature() Signature {
	return Signature{err: xerrors.New("fake error")}
}

// Equal implements crypto.Signature.
func (s Signature) Equal(o crypto.Signature) bool {
	_, ok := o.(Signature)
	return ok
}

// Pack implements encoding.Packable.
func (s Signature) Pack(encoding.ProtoMarshaler) (proto.Message, error) {
	return &wrappers.BytesValue{Value: []byte{SignatureByte}}, s.err
}

// MarshalBinary implements crypto.Signature.
func (s Signature) MarshalBinary() ([]byte, error) {
	return []byte{SignatureByte}, s.err
}

// SignatureFactory is a fake implementation of the signature factory.
type SignatureFactory struct {
	crypto.SignatureFactory
	signature Signature
	err       error
}

// NewSignatureFactory returns a fake signature factory.
func NewSignatureFactory(s Signature) SignatureFactory {
	return SignatureFactory{signature: s}
}

// NewBadSignatureFactory returns a signature factory that will return an error
// when appropriate.
func NewBadSignatureFactory() SignatureFactory {
	return SignatureFactory{err: xerrors.New("fake error")}
}

// FromProto implements crypto.SignatureFactory.
func (f SignatureFactory) FromProto(proto.Message) (crypto.Signature, error) {
	return f.signature, f.err
}

// PublicKey is a fake implementation of crypto.PublicKey.
type PublicKey struct {
	crypto.PublicKey
	err       error
	verifyErr error
}

// NewBadPublicKey returns a new fake public key that returns error when
// appropriate.
func NewBadPublicKey() PublicKey {
	return PublicKey{
		err:       xerrors.New("fake error"),
		verifyErr: xerrors.New("fake error"),
	}
}

// NewInvalidPublicKey returns a fake public key that never verifies.
func NewInvalidPublicKey() PublicKey {
	return PublicKey{verifyErr: xerrors.New("fake error")}
}

// Verify implements crypto.PublicKey.
func (pk PublicKey) Verify([]byte, crypto.Signature) error {
	return pk.verifyErr
}

// MarshalBinary implements encoding.BinaryMarshaler.
func (pk PublicKey) MarshalBinary() ([]byte, error) {
	return []byte{0xdf}, pk.err
}

// Pack implements encoding.Packable.
func (pk PublicKey) Pack(encoding.ProtoMarshaler) (proto.Message, error) {
	return &empty.Empty{}, pk.err
}

// String implements fmt.Stringer.
func (pk PublicKey) String() string {
	return "fake.PublicKey"
}

// Signer is a fake implementation of the crypto.AggregateSigner interface.
type Signer struct {
	crypto.AggregateSigner
	publicKey        PublicKey
	signatureFactory SignatureFactory
	verifierFactory  VerifierFactory
	err              error
}

// NewSigner returns a new instance of the fake signer.
func NewSigner() crypto.AggregateSigner {
	return Signer{}
}

// NewSignerWithSignatureFactory returns a fake signer with the provided
// factory.
func NewSignerWithSignatureFactory(f SignatureFactory) Signer {
	return Signer{signatureFactory: f}
}

// NewSignerWithVerifierFactory returns a new fake signer with the specific
// verifier factory.
func NewSignerWithVerifierFactory(f VerifierFactory) Signer {
	return Signer{verifierFactory: f}
}

// NewSignerWithPublicKey returns a new fake signer with the specific public
// key.
func NewSignerWithPublicKey(k PublicKey) Signer {
	return Signer{publicKey: k}
}

// NewBadSigner returns a fake signer that will return an error when
// appropriate.
func NewBadSigner() Signer {
	return Signer{err: xerrors.New("fake error")}
}

// GetPublicKeyFactory implements crypto.Signer.
func (s Signer) GetPublicKeyFactory() crypto.PublicKeyFactory {
	return PublicKeyFactory{}
}

// GetSignatureFactory implements crypto.Signer.
func (s Signer) GetSignatureFactory() crypto.SignatureFactory {
	return s.signatureFactory
}

// GetVerifierFactory implements crypto.Signer.
func (s Signer) GetVerifierFactory() crypto.VerifierFactory {
	return s.verifierFactory
}

// GetPublicKey implements crypto.Signer.
func (s Signer) GetPublicKey() crypto.PublicKey {
	return s.publicKey
}

// Sign implements crypto.Signer.
func (s Signer) Sign([]byte) (crypto.Signature, error) {
	return Signature{}, s.err
}

// Aggregate implements crypto.AggregateSigner.
func (s Signer) Aggregate(...crypto.Signature) (crypto.Signature, error) {
	return Signature{}, s.err
}

// Verifier is a fake implementation of crypto.Verifier.
type Verifier struct {
	crypto.Verifier
	err error
}

// NewBadVerifier returns a verifier that will return an error when appropriate.
func NewBadVerifier() Verifier {
	return Verifier{err: xerrors.New("fake error")}
}

// Verify implements crypto.Verifier.
func (v Verifier) Verify(msg []byte, s crypto.Signature) error {
	return v.err
}

// VerifierFactory is a fake implementation of crypto.VerifierFactory.
type VerifierFactory struct {
	crypto.VerifierFactory
	verifier Verifier
	err      error
	call     *Call
}

// NewVerifierFactory returns a new fake verifier factory.
func NewVerifierFactory(v Verifier) VerifierFactory {
	return VerifierFactory{verifier: v}
}

// NewVerifierFactoryWithCalls returns a new verifier factory that will register
// the calls.
func NewVerifierFactoryWithCalls(c *Call) VerifierFactory {
	return VerifierFactory{call: c}
}

// NewBadVerifierFactory returns a fake verifier factory that returns an error
// when appropriate.
func NewBadVerifierFactory() VerifierFactory {
	return VerifierFactory{err: xerrors.New("fake error")}
}

// FromAuthority implements crypto.VerifierFactory.
func (f VerifierFactory) FromAuthority(ca crypto.CollectiveAuthority) (crypto.Verifier, error) {
	if f.call != nil {
		f.call.Add(ca)
	}
	return f.verifier, f.err
}

// Counter is a helper to delay errors or actions. It can be nil without panics.
type Counter struct {
	Value int
}

// NewCounter returns a new counter set to the given value.
func NewCounter(value int) *Counter {
	return &Counter{
		Value: value,
	}
}

// Done returns true when the counter reached zero.
func (c *Counter) Done() bool {
	return c == nil || c.Value <= 0
}

// Decrease decrements the counter.
func (c *Counter) Decrease() {
	if c == nil {
		return
	}
	c.Value--
}

// BadPackEncoder is a fake implementation of encoding.ProtoMarshaler.
type BadPackEncoder struct {
	encoding.ProtoEncoder
}

// Pack implements encoding.ProtoMarshaler.
func (e BadPackEncoder) Pack(encoding.Packable) (proto.Message, error) {
	return nil, xerrors.New("fake error")
}

// BadPackAnyEncoder is a fake implementation of encoding.ProtoMarshaler.
type BadPackAnyEncoder struct {
	encoding.ProtoEncoder
	Counter *Counter
}

// PackAny implements encoding.ProtoMarshaler.
func (e BadPackAnyEncoder) PackAny(encoding.Packable) (*any.Any, error) {
	defer e.Counter.Decrease()
	if !e.Counter.Done() {
		return &any.Any{}, nil
	}
	return nil, xerrors.New("fake error")
}

// BadMarshalAnyEncoder is a fake implementation of encoding.ProtoMarshaler.
type BadMarshalAnyEncoder struct {
	encoding.ProtoEncoder
}

// MarshalAny implements encoding.ProtoMarshaler.
func (e BadMarshalAnyEncoder) MarshalAny(proto.Message) (*any.Any, error) {
	return nil, xerrors.New("fake error")
}

// BadUnmarshalAnyEncoder is a fake implementation of encoding.ProtoMarshaler.
type BadUnmarshalAnyEncoder struct {
	encoding.ProtoEncoder
}

// UnmarshalAny implements encoding.ProtoMarshaler.
func (e BadUnmarshalAnyEncoder) UnmarshalAny(*any.Any, proto.Message) error {
	return xerrors.New("fake error")
}

// BadUnmarshalDynEncoder is a fake implementation of encoding.ProtoMarshaler.
type BadUnmarshalDynEncoder struct {
	encoding.ProtoEncoder
}

// UnmarshalDynamicAny implements encoding.ProtoMarshaler.
func (e BadUnmarshalDynEncoder) UnmarshalDynamicAny(*any.Any) (proto.Message, error) {
	return nil, xerrors.New("fake error")
}

// BadMarshalStableEncoder is a fake implementation of encoding.ProtoMarshaler.
type BadMarshalStableEncoder struct {
	encoding.ProtoEncoder
}

// MarshalStable implements encoding.ProtoMarshaler.
func (e BadMarshalStableEncoder) MarshalStable(io.Writer, proto.Message) error {
	return xerrors.New("fake error")
}

// AddressFactory is a fake implementation of mino.AddressFactory.
type AddressFactory struct {
	mino.AddressFactory
}

// FromText implements mino.AddressFactory.
func (f AddressFactory) FromText(text []byte) mino.Address {
	if len(text) >= 4 {
		index := binary.LittleEndian.Uint32(text)
		return Address{index: int(index)}
	}
	return Address{}
}

// Receiver is a fake RPC stream receiver.
type Receiver struct {
	mino.Receiver
	err error
	Msg proto.Message
}

// NewBadReceiver returns a new receiver that returns an error.
func NewBadReceiver() Receiver {
	return Receiver{err: xerrors.New("fake error")}
}

// Recv implements mino.Receiver.
func (r Receiver) Recv(context.Context) (mino.Address, proto.Message, error) {
	return nil, r.Msg, r.err
}

// Sender is a fake RPC stream sender.
type Sender struct {
	mino.Sender
	err error
}

// NewBadSender returns a sender that always returns an error.
func NewBadSender() Sender {
	return Sender{err: xerrors.New("fake error")}
}

// Send implements mino.Sender.
func (s Sender) Send(proto.Message, ...mino.Address) <-chan error {
	errs := make(chan error, 1)
	errs <- s.err
	close(errs)
	return errs
}

// RPC is a fake implementation of mino.RPC.
type RPC struct {
	mino.RPC
	Msgs     chan proto.Message
	Errs     chan error
	receiver Receiver
	sender   Sender
	err      error
}

// NewRPC returns a fake rpc.
func NewRPC() *RPC {
	rpc := &RPC{}
	rpc.Reset()
	return rpc
}

// NewStreamRPC returns a fake rpc with specific stream options.
func NewStreamRPC(r Receiver, s Sender) *RPC {
	rpc := &RPC{
		receiver: r,
		sender:   s,
	}
	rpc.Reset()
	return rpc
}

// NewBadStreamRPC returns a fake rpc that returns an error when calling Stream.
func NewBadStreamRPC() *RPC {
	return &RPC{
		Msgs: make(chan proto.Message, 100),
		Errs: make(chan error, 100),
		err:  xerrors.New("fake error"),
	}
}

// Call implements mino.RPC.
func (rpc *RPC) Call(ctx context.Context, m proto.Message,
	p mino.Players) (<-chan proto.Message, <-chan error) {

	return rpc.Msgs, rpc.Errs
}

// Stream implements mino.RPC.
func (rpc *RPC) Stream(context.Context, mino.Players) (mino.Sender, mino.Receiver, error) {
	return rpc.sender, rpc.receiver, rpc.err
}

// Reset resets the channels.
func (rpc *RPC) Reset() {
	rpc.Msgs = make(chan proto.Message, 100)
	rpc.Errs = make(chan error, 100)
}

// Mino is a fake implementation of mino.Mino.
type Mino struct {
	mino.Mino
	err error
}

// NewBadMino returns a Mino instance that returns an error when appropriate.
func NewBadMino() Mino {
	return Mino{err: xerrors.New("fake error")}
}

// GetAddress implements mino.Mino.
func (m Mino) GetAddress() mino.Address {
	return Address{}
}

// GetAddressFactory implements mino.Mino.
func (m Mino) GetAddressFactory() mino.AddressFactory {
	return AddressFactory{}
}

// MakeRPC implements mino.Mino.
func (m Mino) MakeRPC(string, mino.Handler) (mino.RPC, error) {
	return NewRPC(), m.err
}

// Hash is a fake implementation of hash.Hash.
type Hash struct {
	hash.Hash
	delay int
	err   error
	Call  *Call
}

// NewBadHash returns a fake hash that returns an error when appropriate.
func NewBadHash() *Hash {
	return &Hash{err: xerrors.New("fake error")}
}

// NewBadHashWithDelay returns a fake hash that returns an error after a certain
// amount of calls.
func NewBadHashWithDelay(delay int) *Hash {
	return &Hash{err: xerrors.New("fake error"), delay: delay}
}

func (h *Hash) Write(in []byte) (int, error) {
	if h.Call != nil {
		h.Call.Add(in)
	}

	if h.delay > 0 {
		h.delay--
		return 0, nil
	}
	return 0, h.err
}

// Sum implements hash.Hash.
func (h *Hash) Sum([]byte) []byte {
	return []byte{}
}

// HashFactory is a fake implementation of crypto.HashFactory.
type HashFactory struct {
	hash *Hash
}

// NewHashFactory returns a fake hash factory.
func NewHashFactory(h *Hash) HashFactory {
	return HashFactory{hash: h}
}

// New implements crypto.HashFactory.
func (f HashFactory) New() hash.Hash {
	return f.hash
}
