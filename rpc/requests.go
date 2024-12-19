package rpc

import (
	"context"
	sdkmath "cosmossdk.io/math"
	distributionTypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	stakingTypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"time"

	coretypes "github.com/cometbft/cometbft/rpc/core/types"

	probeClient "github.com/DefiantLabs/probe/client"
	probeQuery "github.com/DefiantLabs/probe/query"
	"github.com/cosmos/cosmos-sdk/types/query"
	txTypes "github.com/cosmos/cosmos-sdk/types/tx"
	"mtt-indexer/logger"
)

// GetBlockTimestamp
func GetBlock(cl *probeClient.ChainClient, height int64) (*coretypes.ResultBlock, error) {
	options := probeQuery.QueryOptions{Height: height}
	query := probeQuery.Query{Client: cl, Options: &options}
	resp, err := query.Block()
	if err != nil {
		return nil, err
	}

	return resp, nil
}

// GetTxsByBlockHeight makes a request to the Cosmos RPC API and returns all the transactions for a specific block
func GetTxsByBlockHeight(cl *probeClient.ChainClient, height int64) (*txTypes.GetTxsEventResponse, error) {
	pg := query.PageRequest{Limit: 100}
	options := probeQuery.QueryOptions{Height: height, Pagination: &pg}
	query := probeQuery.Query{Client: cl, Options: &options}
	resp, err := query.TxByHeight(cl.Codec)
	if err != nil {
		return nil, err
	}

	// handle pagination if needed
	if resp != nil && resp.Pagination != nil {
		// if there are more total objects than we have so far, keep going
		for resp.Pagination.Total > uint64(len(resp.Txs)) {
			query.Options.Pagination.Offset = uint64(len(resp.Txs))
			chunkResp, err := query.TxByHeight(cl.Codec)
			if err != nil {
				return nil, err
			}
			resp.Txs = append(resp.Txs, chunkResp.Txs...)
			resp.TxResponses = append(resp.TxResponses, chunkResp.TxResponses...)
		}
	}

	return resp, nil
}

func GetValidatorReward(cl *probeClient.ChainClient, validatorAddr string) (sdkmath.Int, error) {
	client := distributionTypes.NewQueryClient(cl)
	info, err := client.ValidatorOutstandingRewards(context.Background(), &distributionTypes.QueryValidatorOutstandingRewardsRequest{
		ValidatorAddress: validatorAddr,
	})
	if err != nil {
		return sdkmath.NewInt(0), err
	}
	coin, _ := info.Rewards.Rewards.TruncateDecimal()
	return coin.AmountOfNoDenomValidation("amtt"), nil
}

func AllValidator(cl *probeClient.ChainClient) ([]string, error) {
	client := stakingTypes.NewQueryClient(cl)
	res, err := client.Validators(context.Background(), &stakingTypes.QueryValidatorsRequest{})
	if err != nil {
		return nil, err
	}
	validators := []string{}
	for _, v := range res.Validators {
		validators = append(validators, v.OperatorAddress)
	}
	for uint64(len(validators)) < res.Pagination.Total {
		key := res.Pagination.NextKey
		res, err = client.Validators(context.Background(), &stakingTypes.QueryValidatorsRequest{
			Pagination: &query.PageRequest{
				Key: key,
			},
		})
		if err != nil {
			return nil, err
		}
		for _, v := range res.Validators {
			validators = append(validators, v.OperatorAddress)
		}
	}
	return validators, nil
}

// IsCatchingUp true if the node is catching up to the chain, false otherwise
func IsCatchingUp(cl *probeClient.ChainClient) (bool, error) {
	query := probeQuery.Query{Client: cl, Options: &probeQuery.QueryOptions{}}
	ctx, cancel := query.GetQueryContext()
	defer cancel()

	resStatus, err := query.Client.RPCClient.Status(ctx)
	if err != nil {
		return false, err
	}
	return resStatus.SyncInfo.CatchingUp, nil
}

func GetLatestBlockHeight(cl *probeClient.ChainClient) (int64, error) {
	query := probeQuery.Query{Client: cl, Options: &probeQuery.QueryOptions{}}
	ctx, cancel := query.GetQueryContext()
	defer cancel()

	resStatus, err := query.Client.RPCClient.Status(ctx)
	if err != nil {
		return 0, err
	}
	return resStatus.SyncInfo.LatestBlockHeight, nil
}

func GetLatestBlockHeightWithRetry(cl *probeClient.ChainClient, retryMaxAttempts int64, retryMaxWaitSeconds uint64) (int64, error) {
	if retryMaxAttempts == 0 {
		return GetLatestBlockHeight(cl)
	}

	if retryMaxWaitSeconds < 2 {
		retryMaxWaitSeconds = 2
	}

	var attempts int64
	maxRetryTime := time.Duration(retryMaxWaitSeconds) * time.Second
	if maxRetryTime < 0 {
		logger.Logger.Warn("Detected maxRetryTime overflow, setting time to sane maximum of 30s")
		maxRetryTime = 30 * time.Second
	}

	currentBackoffDuration, maxReached := GetBackoffDurationForAttempts(attempts, maxRetryTime)

	for {
		resp, err := GetLatestBlockHeight(cl)
		attempts++
		if err != nil && (retryMaxAttempts < 0 || (attempts <= retryMaxAttempts)) {
			logger.Logger.Error("Error getting RPC response, backing off and trying again", err)
			logger.Logger.Debugf("Attempt %d with wait time %+v", attempts, currentBackoffDuration)
			time.Sleep(currentBackoffDuration)

			// guard against overflow
			if !maxReached {
				currentBackoffDuration, maxReached = GetBackoffDurationForAttempts(attempts, maxRetryTime)
			}

		} else {
			if err != nil {
				logger.Logger.Error("Error getting RPC response, reached max retry attempts")
			}
			return resp, err
		}
	}
}

func GetEarliestAndLatestBlockHeights(cl *probeClient.ChainClient) (int64, int64, error) {
	query := probeQuery.Query{Client: cl, Options: &probeQuery.QueryOptions{}}
	ctx, cancel := query.GetQueryContext()
	defer cancel()

	resStatus, err := query.Client.RPCClient.Status(ctx)
	if err != nil {
		return 0, 0, err
	}
	return resStatus.SyncInfo.EarliestBlockHeight, resStatus.SyncInfo.LatestBlockHeight, nil
}
