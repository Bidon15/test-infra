package nodekit

import (
	"fmt"
	"net"

	"github.com/celestiaorg/celestia-app/app"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/celestiaorg/celestia-node/logs"
	"github.com/celestiaorg/celestia-node/node"
	"github.com/celestiaorg/celestia-node/params"
	logging "github.com/ipfs/go-log/v2"

	"go.uber.org/fx"
)

func NewConfig(
	tp node.Type,
	IP net.IP,
	trustedPeers []string,
	trustedHash string,
) *node.Config {
	cfg := node.DefaultConfig(tp)
	cfg.P2P.ListenAddresses = []string{fmt.Sprintf("/ip4/%s/tcp/2121", IP)}
	cfg.Header.TrustedPeers = trustedPeers
	cfg.Header.TrustedHash = trustedHash

	return cfg
}

func NewNode(
	path string,
	tp node.Type,
	cfg *node.Config,
	options ...fx.Option,
) (*node.Node, error) {
	// This is necessary to ensure that the account addresses are correctly prefixed
	// as in the celestia application.
	sdkcfg := sdk.GetConfig()
	sdkcfg.SetBech32PrefixForAccount(app.Bech32PrefixAccAddr, app.Bech32PrefixAccPub)
	sdkcfg.SetBech32PrefixForValidator(app.Bech32PrefixValAddr, app.Bech32PrefixValPub)
	sdkcfg.Seal()

	err := node.Init(*cfg, path, tp)
	if err != nil {
		return nil, err
	}
	store, err := node.OpenStore(path)
	if err != nil {
		return nil, err
	}

	options = append([]fx.Option{node.WithNetwork(params.Private)}, options...)
	return node.New(tp, store, options...)
}

func SetLoggersLevel(lvl string) error {
	level, err := logging.LevelFromString(lvl)
	if err != nil {
		return err
	}
	logs.SetAllLoggers(level)

	return nil
}
