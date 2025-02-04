package price_encoder

import (
	"fmt"
	"github.com/cometbft/cometbft/libs/log"
	pf_constants "github.com/dydxprotocol/v4-chain/protocol/daemons/pricefeed/client/constants"
	"github.com/dydxprotocol/v4-chain/protocol/daemons/pricefeed/client/price_fetcher"
	"github.com/dydxprotocol/v4-chain/protocol/daemons/pricefeed/client/types"
	pft "github.com/dydxprotocol/v4-chain/protocol/daemons/pricefeed/types"
	"github.com/dydxprotocol/v4-chain/protocol/testutil/constants"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func generateBufferedChannelAndExchangeToMarketPrices(
	t *testing.T,
	exchangeIds []types.ExchangeId,
) (
	types.ExchangeToMarketPrices,
	chan *price_fetcher.PriceFetcherSubtaskResponse,
) {
	etmp, err := types.NewExchangeToMarketPrices(exchangeIds)
	require.NoError(t, err)

	bCh := make(chan *price_fetcher.PriceFetcherSubtaskResponse, pf_constants.FixedBufferSize)

	return etmp.(*types.ExchangeToMarketPricesImpl), bCh
}

func genNewPriceEncoder(t *testing.T) *PriceEncoderImpl {
	etmp, bCh := generateBufferedChannelAndExchangeToMarketPrices(t, []types.ExchangeId{constants.ExchangeId1})
	pe, err := NewPriceEncoder(
		&constants.Exchange1_3Markets_MutableExchangeMarketConfig,
		constants.MutableMarketConfigs_3Markets,
		etmp,
		log.TestingLogger(),
		bCh,
	)
	require.NoError(t, err)
	return pe
}

func TestGetExchangeId(t *testing.T) {
	// 1. Setup
	pe := genNewPriceEncoder(t)

	// 2. Test
	require.Equal(t, constants.ExchangeId1, pe.GetExchangeId())
}

func TestUpdateMutableExchangeConfig_Mixed(t *testing.T) {
	tests := map[string]struct {
		updateExchangeConfig *types.MutableExchangeMarketConfig
		updateMarketConfigs  []*types.MutableMarketConfig
		expectedError        error
	}{
		"Failed - Exchange ID mismatch": {
			updateExchangeConfig: &types.MutableExchangeMarketConfig{
				Id: constants.ExchangeId2,
			},
			updateMarketConfigs: constants.MutableMarketConfigs_3Markets,
			expectedError: fmt.Errorf(
				"PriceEncoder.UpdateMutableExchangeConfig: exchange id mismatch, expected '%v', got '%v'",
				constants.ExchangeId1,
				constants.ExchangeId2,
			),
		},
		"Failed - Invalid config": {
			updateExchangeConfig: &constants.Exchange1_1Markets_MutableExchangeMarketConfig,
			updateMarketConfigs:  []*types.MutableMarketConfig{},
			expectedError: fmt.Errorf(
				"PriceEncoder.UpdateMutableExchangeConfig: invalid exchange config update: no market config " +
					"for market 7 on exchange 'Exchange1'"),
		},
		"Success": {
			updateExchangeConfig: &constants.Exchange1_5Markets_MutableExchangeMarketConfig,
			updateMarketConfigs:  constants.MutableMarketConfigs_5Markets,
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			pe := genNewPriceEncoder(t)
			err := pe.UpdateMutableExchangeConfig(tc.updateExchangeConfig, tc.updateMarketConfigs)

			if tc.expectedError != nil {
				require.EqualError(t, err, tc.expectedError.Error())
			} else {
				require.NoError(t, err)
			}
		})
	}
}

type MockExchangeToMarketPrices struct {
	types.ExchangeToMarketPrices
	indexPrice          uint64
	numPricesMedianized int
}

func (m *MockExchangeToMarketPrices) GetIndexPrice(types.MarketId, time.Time, pft.Resolver) (uint64, int) {
	return m.indexPrice, m.numPricesMedianized
}

func TestConvertPriceUpdate_Mixed(t *testing.T) {
	tests := map[string]struct {
		mutableExchangeConfig               *types.MutableExchangeMarketConfig
		mutableMarketConfigs                []*types.MutableMarketConfig
		adjustmentMarketIndexPrice          uint64
		adjustmentMarketNumPricesMedianized int
		expectedPrice                       uint64
		expectedErr                         error
	}{
		"Success - no conversion": {
			mutableExchangeConfig: &types.MutableExchangeMarketConfig{
				Id: constants.ExchangeId1,
				MarketToMarketConfig: map[types.MarketId]types.MarketConfig{
					1: {
						Ticker: "PAIR-USD",
					},
				},
			},
			mutableMarketConfigs: []*types.MutableMarketConfig{
				{
					Id:           1,
					Pair:         "PAIR-USD",
					Exponent:     -6,
					MinExchanges: 1,
				},
			},
			expectedPrice: constants.FiveBillion,
		},
		"Success - inverted price": {
			mutableExchangeConfig: &types.MutableExchangeMarketConfig{
				Id: constants.ExchangeId1,
				MarketToMarketConfig: map[types.MarketId]types.MarketConfig{
					1: {
						Ticker: "PAIR-USD",
						Invert: true,
					},
				},
			},
			mutableMarketConfigs: []*types.MutableMarketConfig{
				{
					Id:           1,
					Pair:         "PAIR-USD",
					Exponent:     -10,
					MinExchanges: 1,
				},
			},
			expectedPrice: uint64(20_000_000_000),
		},
		"Success - division with adjust-by market": {
			mutableExchangeConfig: &types.MutableExchangeMarketConfig{
				Id: constants.ExchangeId1,
				MarketToMarketConfig: map[types.MarketId]types.MarketConfig{
					1: {
						Ticker:         "PAIR-USD",
						AdjustByMarket: newMarketIdWithValue(2),
						Invert:         true,
					},
				},
			},
			mutableMarketConfigs: []*types.MutableMarketConfig{
				{
					Id:           1,
					Pair:         "PAIR-USD",
					Exponent:     -6,
					MinExchanges: 1,
				},
				{
					Id:           2,
					Pair:         "ADJ-USD",
					Exponent:     -10,
					MinExchanges: 1,
				},
			},
			adjustmentMarketIndexPrice:          constants.FiveBillion * 15_000, // 1.5x price.
			adjustmentMarketNumPricesMedianized: 1,
			expectedPrice:                       uint64(1_500_000), // Expect 1.5e6.
		},
		"Success - multiplication with adjust-by market": {
			mutableExchangeConfig: &types.MutableExchangeMarketConfig{
				Id: constants.ExchangeId1,
				MarketToMarketConfig: map[types.MarketId]types.MarketConfig{
					1: {
						Ticker:         "PAIR-USD",
						AdjustByMarket: newMarketIdWithValue(2),
					},
				},
			},
			mutableMarketConfigs: []*types.MutableMarketConfig{
				{
					Id:           1,
					Pair:         "PAIR-USD",
					Exponent:     -6,
					MinExchanges: 1,
				},
				{
					Id:           2,
					Pair:         "ADJ-USD",
					Exponent:     -9,
					MinExchanges: 1,
				},
			},
			adjustmentMarketIndexPrice:          uint64(990_000_000), // 0.99.
			adjustmentMarketNumPricesMedianized: 1,
			expectedPrice:                       uint64(4_950_000_000), // 5 billion * 99%.
		},
		"Failure - invalid index price": {
			mutableExchangeConfig: &types.MutableExchangeMarketConfig{
				Id: constants.ExchangeId1,
				MarketToMarketConfig: map[types.MarketId]types.MarketConfig{
					1: {
						Ticker:         "PAIR-USD",
						AdjustByMarket: newMarketIdWithValue(2),
					},
				},
			},
			mutableMarketConfigs: []*types.MutableMarketConfig{
				{
					Id:           1,
					Pair:         "PAIR-USD",
					Exponent:     -6,
					MinExchanges: 1,
				},
				{
					Id:           2,
					Pair:         "ADJ-USD",
					Exponent:     -9,
					MinExchanges: 2,
				},
			},
			adjustmentMarketIndexPrice:          uint64(990_000_000),
			adjustmentMarketNumPricesMedianized: 1, // Should be at least 2.
			expectedErr: fmt.Errorf(
				"Could not retrieve index price for market 2: expected median price from 2 exchanges, but got " +
					"1 exchanges)",
			),
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			emtp := MockExchangeToMarketPrices{
				indexPrice:          tc.adjustmentMarketIndexPrice,
				numPricesMedianized: tc.adjustmentMarketNumPricesMedianized,
			}
			pe, err := NewPriceEncoder(
				tc.mutableExchangeConfig,
				tc.mutableMarketConfigs,
				&emtp,
				log.TestingLogger(),
				nil,
			)
			require.NoError(t, err)
			convertedPriceTimestamp, err := pe.convertPriceUpdate(
				&types.MarketPriceTimestamp{
					MarketId:      constants.MarketId1,
					Price:         constants.FiveBillion,
					LastUpdatedAt: constants.TimeT,
				},
			)
			if tc.expectedErr != nil {
				require.Error(t, tc.expectedErr, err.Error())
				require.Zero(t, convertedPriceTimestamp)
			} else {
				require.NoError(t, err)
				require.Equal(t, constants.TimeT, convertedPriceTimestamp.LastUpdatedAt)
				require.Equal(t, constants.MarketId1, convertedPriceTimestamp.MarketId)
				require.Equal(t, tc.expectedPrice, convertedPriceTimestamp.Price)
			}
		})
	}
}
