package controller

import (
	"go.dedis.ch/dela"
	"go.dedis.ch/dela/cli/node"
	"go.dedis.ch/dela/core/ordering/cosipbft/authority"
	"go.dedis.ch/dela/crypto"
	"go.dedis.ch/dela/dkg"
	"go.dedis.ch/dela/mino"
	"golang.org/x/xerrors"
)

type initAction struct {
}

func (a *initAction) Execute(ctx node.Context) error {
	var dkgPedersen dkg.DKG
	err := ctx.Injector.Resolve(&dkgPedersen)
	if err != nil {
		return xerrors.Errorf("failed to resolve dkg: %v", err)
	}

	actor, err := dkgPedersen.Listen()
	if err != nil {
		return xerrors.Errorf("failed to start the RPC: %v", err)
	}

	ctx.Injector.Inject(actor)
	dela.Logger.Info().Msg("DKG has been initialized successfully")
	return nil
}

type setupAction struct {
}

func (a *setupAction) Execute(ctx node.Context) error {
	var actor dkg.Actor
	err := ctx.Injector.Resolve(&actor)
	if err != nil {
		return xerrors.Errorf("failed to resolve actor: %v", err)
	}

	//addrs := actor.(*pedersen.Actor)

	addrs := make([]mino.Address, 5)
	pubkeys := make([]crypto.PublicKey, 5)

	ca := authority.New(addrs, pubkeys)

	pubkey, err := actor.Setup(ca, ca.Len())
	if err != nil {
		return xerrors.Errorf("failed to setup DKG: %v", err)
	}

	pubkeyBuf, err := pubkey.MarshalBinary()
	if err != nil {
		return xerrors.Errorf("failed to encode pubkey: %v", err)
	}

	dela.Logger.Info().
		Hex("DKG public key", pubkeyBuf).
		Msg("DKG public key")

	return nil
}

type getPublicKeyAction struct {
}

func (a *getPublicKeyAction) Execute(ctx node.Context) error {
	var actor dkg.Actor
	err := ctx.Injector.Resolve(&actor)
	if err != nil {
		return xerrors.Errorf("failed to resolve actor: %v", err)
	}

	pubkey, err := actor.GetPublicKey()
	if err != nil {
		return xerrors.Errorf("failed to retrieve the public key: %v", err)
	}

	pubkeyBuf, err := pubkey.MarshalBinary()
	if err != nil {
		return xerrors.Errorf("failed to encode pubkey: %v", err)
	}

	dela.Logger.Info().
		Hex("DKG public key", pubkeyBuf).
		Msg("DKG public key")

	return nil
}

type encryptAction struct {
}

func (a *encryptAction) Execute(ctx node.Context) error {
	return nil
}

type decryptAction struct {
}

func (a *decryptAction) Execute(ctx node.Context) error {
	return nil
}
