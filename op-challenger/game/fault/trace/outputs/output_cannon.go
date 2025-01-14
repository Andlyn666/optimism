package outputs

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/ethereum-optimism/optimism/op-challenger/config"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/contracts"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace/cannon"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace/split"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	"github.com/ethereum-optimism/optimism/op-challenger/metrics"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
)

func NewOutputCannonTraceAccessor(
	ctx context.Context,
	logger log.Logger,
	m metrics.Metricer,
	cfg *config.Config,
	contract *contracts.FaultDisputeGameContract,
	dir string,
	gameDepth uint64,
	prestateBlock uint64,
	poststateBlock uint64,
) (*trace.Accessor, error) {
	// TODO(client-pod#43): Load depths from the contract
	topDepth := gameDepth / 2
	bottomDepth := gameDepth - topDepth
	outputProvider, err := NewTraceProvider(ctx, logger, cfg.RollupRpc, topDepth, prestateBlock, poststateBlock)
	if err != nil {
		return nil, err
	}

	cannonCreator := func(ctx context.Context, localContext common.Hash, agreed contracts.Proposal, claimed contracts.Proposal) (types.TraceProvider, error) {
		logger := logger.New("pre", agreed.OutputRoot, "post", claimed.OutputRoot, "localContext", localContext)
		subdir := filepath.Join(dir, localContext.Hex())
		provider, err := cannon.NewTraceProviderFromProposals(ctx, logger, m, cfg, contract, localContext, agreed, claimed, subdir, bottomDepth)
		if err != nil {
			return nil, fmt.Errorf("failed to create cannon trace provider: %w", err)
		}
		return provider, nil
	}

	cache := NewProviderCache(m, "output_cannon_provider", cannonCreator)
	selector := split.NewSplitProviderSelector(outputProvider, int(topDepth), OutputRootSplitAdapter(outputProvider, cache.GetOrCreate))
	return trace.NewAccessor(selector), nil
}
