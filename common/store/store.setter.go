package store

import (
	"context"
	"encoding/json"
	"strconv"
	"sync"

	"github.com/dgraph-io/badger/v3"
	"github.com/ethereum/go-ethereum/common"
	"github.com/yearn/ydaemon/common/bigNumber"
	"github.com/yearn/ydaemon/common/helpers"
	"github.com/yearn/ydaemon/common/logs"
	"github.com/yearn/ydaemon/internal/models"
	"golang.org/x/time/rate"
)

var storeRateLimiter = rate.NewLimiter(2, 4)

/**************************************************************************************************
** StoreBlockTime will store the blockTime in the _blockTimeSyncMap for fast access during that
** same execution, and will store it in the configured DB for future executions.
**************************************************************************************************/
func StoreBlockTime(chainID uint64, blockNumber uint64, blockTime uint64) {
	syncMap := _blockTimeSyncMap[chainID]
	if syncMap == nil {
		syncMap = &sync.Map{}
		_blockTimeSyncMap[chainID] = syncMap
	}
	syncMap.Store(blockNumber, blockTime)

	logs.Info(`Storing block time for chain ` + strconv.FormatUint(chainID, 10) + ` block ` + strconv.FormatUint(blockNumber, 10) + ` time ` + strconv.FormatUint(blockTime, 10))
	switch _dbType {
	case DBBadger:
		go OpenBadgerDB(chainID, TABLES.BLOCK_TIME).Update(func(txn *badger.Txn) error {
			dataByte, err := json.Marshal(blockTime)
			if err != nil {
				return err
			}
			return txn.Set([]byte(strconv.FormatUint(blockNumber, 10)), dataByte)
		})
	case DBMysql:
		go func() {
			DBbaseSchema := DBBaseSchema{
				UUID:    getUUID(strconv.FormatUint(chainID, 10) + strconv.FormatUint(blockNumber, 10) + strconv.FormatUint(blockTime, 10)),
				Block:   blockNumber,
				ChainID: chainID,
			}
			storeRateLimiter.Wait(context.Background())
			DATABASE.Table(`db_block_times`).Save(&DBBlockTime{DBbaseSchema, blockTime})
		}()
	}
}

/**************************************************************************************************
** StoreHistoricalPrice will store the price of a token at a specific block in the
** _historicalPriceSyncMap for fast access during that same execution, and will store it in the
** configured DB for future executions.
**************************************************************************************************/
func StoreHistoricalPrice(chainID uint64, blockNumber uint64, tokenAddress common.Address, price *bigNumber.Int) {
	syncMap := _historicalPriceSyncMap[chainID]
	key := strconv.FormatUint(blockNumber, 10) + "_" + tokenAddress.Hex()
	syncMap.Store(key, price)

	switch _dbType {
	case DBBadger:
		go OpenBadgerDB(chainID, TABLES.HISTORICAL_PRICES).Update(func(txn *badger.Txn) error {
			dataByte, err := json.Marshal(price.String())
			if err != nil {
				return err
			}
			return txn.Set([]byte(key), dataByte)
		})
	case DBMysql:
		go func() {
			DBbaseSchema := DBBaseSchema{
				UUID:    getUUID(strconv.FormatUint(chainID, 10) + strconv.FormatUint(blockNumber, 10) + tokenAddress.Hex()),
				Block:   blockNumber,
				ChainID: chainID,
			}
			humanizedPrice, _ := helpers.ToNormalizedAmount(price, 6).Float64()
			storeRateLimiter.Wait(context.Background())
			DATABASE.Table(`db_historical_prices`).Save(&DBHistoricalPrice{
				DBbaseSchema,
				tokenAddress.Hex(),
				price.String(),
				humanizedPrice,
			})
		}()
	}
}

/**************************************************************************************************
** StoreNewVaultsFromRegistry will store a new vault in the _newVaultsFromRegistrySyncMap for fast
** access during that same execution, and will store it in the configured DB for future executions.
**************************************************************************************************/
func StoreNewVaultsFromRegistry(chainID uint64, vault models.TVaultsFromRegistry) {
	syncMap := _newVaultsFromRegistrySyncMap[chainID]
	key := strconv.FormatUint(vault.BlockNumber, 10) + "_" + vault.RegistryAddress.Hex() + "_" + vault.Address.Hex() + "_" + vault.TokenAddress.Hex() + "_" + vault.APIVersion
	syncMap.Store(key, vault)

	switch _dbType {
	case DBBadger:
		// Not implemented
	case DBMysql:
		go func() {
			DBbaseSchema := DBBaseSchema{
				UUID:    getUUID(key),
				Block:   vault.BlockNumber,
				ChainID: chainID,
			}
			storeRateLimiter.Wait(context.Background())
			DATABASE.Table(`db_new_vaults_from_registries`).Save(&DBNewVaultsFromRegistry{
				DBbaseSchema,
				vault.RegistryAddress.Hex(),
				vault.Address.Hex(),
				vault.TokenAddress.Hex(),
				vault.BlockHash.Hex(),
				vault.Type,
				vault.APIVersion,
				vault.Activation,
				vault.ManagementFee,
				vault.TxIndex,
				vault.LogIndex,
			})
		}()
	}
}

/**************************************************************************************************
** StoreVault will store a new vault in the _vaultsSyncMap for fast access during that same
** execution, and will store it in the configured DB for future executions.
**************************************************************************************************/
func StoreVault(chainID uint64, vault *models.TVault) {
	syncMap := _vaultsSyncMap[chainID]
	key := vault.Address.Hex() + "_" + vault.Token.Address.Hex() + "_" + strconv.FormatUint(vault.Activation, 10) + "_" + strconv.FormatUint(vault.ChainID, 10)
	syncMap.Store(vault.Address, vault)

	switch _dbType {
	case DBBadger:
		go OpenBadgerDB(chainID, TABLES.VAULTS).Update(func(txn *badger.Txn) error {
			dataByte, err := json.Marshal(vault)
			if err != nil {
				return err
			}
			return txn.Set([]byte(key), dataByte)
		})
	case DBMysql:
		//for now
		go OpenBadgerDB(chainID, TABLES.VAULTS).Update(func(txn *badger.Txn) error {
			dataByte, err := json.Marshal(vault)
			if err != nil {
				return err
			}
			return txn.Set([]byte(key), dataByte)
		})
		go func() {
			newItem := &DBVault{}
			newItem.UUID = getUUID(key)
			newItem.Address = vault.Address.Hex()
			newItem.Management = vault.Management.Hex()
			newItem.Governance = vault.Governance.Hex()
			newItem.Guardian = vault.Guardian.Hex()
			newItem.Rewards = vault.Rewards.Hex()
			newItem.Token = vault.Token.Address.Hex()
			newItem.Type = vault.Type
			newItem.Symbol = vault.Symbol
			newItem.DisplaySymbol = vault.DisplaySymbol
			newItem.FormatedSymbol = vault.FormatedSymbol
			newItem.Name = vault.Name
			newItem.DisplayName = vault.DisplayName
			newItem.FormatedName = vault.FormatedName
			newItem.Icon = vault.Icon
			newItem.Version = vault.Version
			newItem.ChainID = vault.ChainID
			newItem.Inception = vault.Inception
			newItem.Activation = vault.Activation
			newItem.Decimals = vault.Decimals
			newItem.PerformanceFee = vault.PerformanceFee
			newItem.ManagementFee = vault.ManagementFee
			newItem.Endorsed = vault.Endorsed
			storeRateLimiter.Wait(context.Background())
			DATABASE.Table(`db_vaults`).Save(newItem)
		}()
	}
}