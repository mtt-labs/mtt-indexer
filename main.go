package main

import (
	"flag"
	"fmt"
	sdkTypes "github.com/cosmos/cosmos-sdk/types"
	"mtt-indexer/config"
	"mtt-indexer/cornjob"
	"mtt-indexer/db"
	"mtt-indexer/filter"
	"mtt-indexer/logger"
	"mtt-indexer/parsers"
	"mtt-indexer/router"
	"mtt-indexer/service"
	"mtt-indexer/types"
	"mtt-indexer/util"
	"net/http"
	"os"
	"sync"
)

var (
	configFlag = flag.String("config", "config.yaml", "Config file")
)

func init() {
	prefix := os.Getenv("ACCOUNT_PREFIX")
	// Set prefixes
	accountPubKeyPrefix := prefix + "pub"
	validatorAddressPrefix := prefix + "valoper"
	validatorPubKeyPrefix := prefix + "valoperpub"
	consNodeAddressPrefix := prefix + "valcons"
	consNodePubKeyPrefix := prefix + "valconspub"

	sdkConfig := sdkTypes.GetConfig()

	sdkConfig.SetCoinType(44)
	sdkConfig.SetPurpose(60)

	sdkConfig.SetBech32PrefixForAccount(prefix, accountPubKeyPrefix)
	sdkConfig.SetBech32PrefixForValidator(validatorAddressPrefix, validatorPubKeyPrefix)
	sdkConfig.SetBech32PrefixForConsensusNode(consNodeAddressPrefix, consNodePubKeyPrefix)
	sdkConfig.Seal()
}

func main() {
	util.LoadConfig(*configFlag, &config.Cfg)
	cfg := &config.Cfg
	db := db.NewLdb(cfg.DbTailFix)

	newService := service.NewService(db)
	engine := router.Init(newService)
	addr := fmt.Sprintf("0.0.0.0:%d", cfg.Port)
	srv := &http.Server{
		Addr:    addr,
		Handler: engine,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil {
			logger.Logger.Fatal("listen addr:%s,err:%v", addr, err)
		}
	}()

	chain := &types.Chain{
		Name: "mtt",
	}
	record, err := db.GetRecordByType(chain)
	if err != nil {
		return
	}
	if record != nil {
		if chainRecord, ok := record.(*types.Chain); ok {
			chain = chainRecord
		}
	}

	cl, err := service.NewChainClient(chain, cfg.Rpc)
	if err != nil {
		logger.Logger.Fatal(err)
	}

	chainService, err := service.NewChainService(db, chain, cl)
	if err != nil {
		logger.Logger.Fatal(err)
	}

	go cornjob.CronJobLedgerInit(db, cl)

	stakingDelegateRegexMessageTypeFilter, err := filter.NewRegexMessageTypeFilter("^/cosmos\\.staking.*MsgDelegate$", false)
	if err != nil {
		logger.Logger.Fatalf("Failed to create regex message type filter. Err: %v", err)
	}

	stakingUndelegateRegexMessageTypeFilter, err := filter.NewRegexMessageTypeFilter("^/cosmos\\.staking.*MsgUndelegate$", false)
	if err != nil {
		logger.Logger.Fatalf("Failed to create regex message type filter. Err: %v", err)
	}

	stakingCreateValidatorTypeFilter, err := filter.NewRegexMessageTypeFilter("^/cosmos\\.staking.*MsgMsgCreateValidator$", false)
	if err != nil {
		logger.Logger.Fatalf("Failed to create regex message type filter. Err: %v", err)
	}

	distributionWithdrawDelegatorFilter, err := filter.NewRegexMessageTypeFilter("^/cosmos\\.distribution.*MsgWithdrawDelegatorReward$", false)
	if err != nil {
		logger.Logger.Fatalf("Failed to create regex message type filter. Err: %v", err)
	}

	stakingCancelUnbondingTypeFilter, err := filter.NewRegexMessageTypeFilter("^/cosmos\\.staking.*MsgCancelUnbondingDelegation$", false)
	if err != nil {
		logger.Logger.Fatalf("Failed to create regex message type filter. Err: %v", err)
	}

	redelegateFilter, err := filter.NewRegexMessageTypeFilter("^/cosmos\\.staking.*MsgBeginRedelegate$", false)
	if err != nil {
		logger.Logger.Fatalf("Failed to create regex message type filter. Err: %v", err)
	}

	distributionWithdrawCommissionFilter, err := filter.NewRegexMessageTypeFilter("^/cosmos\\.distribution.*MsgWithdrawValidatorCommission$", false)
	if err != nil {
		logger.Logger.Fatalf("Failed to create regex message type filter. Err: %v", err)
	}

	chainService.RegisterMessageTypeFilter(stakingDelegateRegexMessageTypeFilter)
	chainService.RegisterMessageTypeFilter(stakingUndelegateRegexMessageTypeFilter)
	chainService.RegisterMessageTypeFilter(stakingCreateValidatorTypeFilter)
	chainService.RegisterMessageTypeFilter(distributionWithdrawDelegatorFilter)
	chainService.RegisterMessageTypeFilter(stakingCancelUnbondingTypeFilter)
	chainService.RegisterMessageTypeFilter(redelegateFilter)
	chainService.RegisterMessageTypeFilter(distributionWithdrawCommissionFilter)

	delegateParser := &parsers.MsgDelegateUndelegateParser{Id: "delegate"}
	undelegateParser := &parsers.MsgDelegateUndelegateParser{Id: "undelegate"}
	createValidatorParser := &parsers.MsgCreateValidatorParser{Id: "validator"}
	withdrawDelegatorRewardParser := &parsers.MsgWithdrawDelegatorRewardParser{Id: "delegatorReward"}
	cancelUnnbondingParser := &parsers.MsgCancelUnbondingParser{Id: "cancelUnbonding"}
	redelegateParser := &parsers.MsgRedelegateParser{Id: "redelegate"}
	withdrawCommissionParser := &parsers.MsgRedelegateParser{Id: "commissionReward"}

	chainService.RegisterCustomMessageParser("/cosmos.staking.v1beta1.MsgDelegate", delegateParser)
	chainService.RegisterCustomMessageParser("/cosmos.staking.v1beta1.MsgUndelegate", undelegateParser)
	chainService.RegisterCustomMessageParser("/cosmos.staking.v1beta1.MsgCreateValidator", createValidatorParser)
	chainService.RegisterCustomMessageParser("/cosmos.distribution.v1beta1.MsgWithdrawDelegatorReward", withdrawDelegatorRewardParser)
	chainService.RegisterCustomMessageParser("/cosmos.staking.v1beta1.MsgCancelUnbondingDelegation", cancelUnnbondingParser)
	chainService.RegisterCustomMessageParser("/cosmos.staking.v1beta1.MsgBeginRedelegate", redelegateParser)
	chainService.RegisterCustomMessageParser("/cosmos.distribution.v1beta1.MsgWithdrawValidatorCommission", withdrawCommissionParser)
	var wg sync.WaitGroup

	wg.Add(1)
	chainService.Start(&wg)

	wg.Wait()
}
