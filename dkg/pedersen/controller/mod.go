package controller

import (
	"go.dedis.ch/dela"
	"go.dedis.ch/dela/cli"
	"go.dedis.ch/dela/cli/node"
	"go.dedis.ch/dela/dkg/pedersen"
	"go.dedis.ch/dela/mino"
	"golang.org/x/xerrors"
)

// NewMinimal returns a new minimal initializer
func NewMinimal() node.Initializer {
	return minimal{}
}

// minimal is an initializer with the minimum set of commands. Indeed it only
// creates and injects a new DKG
//
// - implements node.Initializer
type minimal struct{}

// Build implements node.Initializer.
func (m minimal) SetCommands(builder node.Builder) {
	cmd := builder.SetCommand("dkg")
	cmd.SetDescription("... ")

	sub := cmd.SetSubCommand("init")
	sub.SetDescription("Initialize the DKG protocol")
	sub.SetAction(builder.MakeAction(&initAction{}))

	sub = cmd.SetSubCommand("setup")
	sub.SetDescription("Creates the public distributed key and the private share on each node")
	sub.SetAction(builder.MakeAction(&setupAction{}))

	sub = cmd.SetSubCommand("getPublicKey")
	sub.SetDescription("Prints the public Key")
	sub.SetAction(builder.MakeAction(&getPublicKeyAction{}))

	sub = cmd.SetSubCommand("encrypt")
	sub.SetDescription("Encrypt the given string and write the ciphertext pair in the corresponding file")
	sub.SetFlags(cli.StringFlag{
		Name:     "plaintext",
		Usage:    "plaintext to encrypt",
		Required: true,
	}, cli.StringFlag{
		Name:     "filePath",
		Usage:    "path to write the ciphertext pair",
		Required: true,
	})
	sub.SetAction(builder.MakeAction(&encryptAction{}))

	sub = cmd.SetSubCommand("decrypt")
	sub.SetDescription("Decrypt the given ciphertext pair and print the corresponding plaintext")
	sub.SetFlags(cli.StringFlag{
		Name:     "filePath",
		Usage:    "path to read the ciphertext pair",
		Required: true,
	})
	sub.SetAction(builder.MakeAction(&decryptAction{}))

}

// OnStart implements node.Initializer. It creates and registers a pedersen DKG.
func (m minimal) OnStart(ctx cli.Flags, inj node.Injector) error {
	var no mino.Mino
	err := inj.Resolve(&no)
	if err != nil {
		return xerrors.Errorf("failed to resolve mino: %v", err)
	}

	dkg, pubkey := pedersen.NewPedersen(no)

	inj.Inject(dkg)

	pubkeyBuf, err := pubkey.MarshalBinary()
	if err != nil {
		return xerrors.Errorf("failed to encode pubkey: %v", err)
	}

	dela.Logger.Info().
		Hex("public key", pubkeyBuf).
		Msg("perdersen public key")

	return nil
}

// OnStop implements node.Initializer.
func (minimal) OnStop(node.Injector) error {
	return nil
}
