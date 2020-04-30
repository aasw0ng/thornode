package bitcoin

import (
	"encoding/json"
	"fmt"

	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/util"
)

// PrefixUTXOStorage declares prefix to use in leveldb to avoid conflicts
const (
	TransactionFeeKey = "transactionfee-"
	PrefixBlocMeta    = `blockmeta-`
)

// LevelDBBlockMetaAccessor struct
type LevelDBBlockMetaAccessor struct {
	db *leveldb.DB
}

// NewLevelDBBlockMetaAccessor creates a new level db utxo accessor
func NewLevelDBBlockMetaAccessor(db *leveldb.DB) (*LevelDBBlockMetaAccessor, error) {
	return &LevelDBBlockMetaAccessor{db: db}, nil
}

func (t *LevelDBBlockMetaAccessor) getBlockMetaKey(height int64) string {
	return fmt.Sprintf(PrefixBlocMeta+"%d", height)
}

// GetBlockMeta at given block height
func (t *LevelDBBlockMetaAccessor) GetBlockMeta(height int64) (*BlockMeta, error) {
	key := t.getBlockMetaKey(height)
	exist, err := t.db.Has([]byte(key), nil)
	if err != nil {
		return nil, fmt.Errorf("fail to check whether block meta(%s) exist: %w", key, err)
	}
	if !exist {
		return nil, nil
	}
	v, err := t.db.Get([]byte(key), nil)
	if err != nil {
		return nil, fmt.Errorf("fail to get block meta(%s) from storage: %w", key, err)
	}
	var blockMeta BlockMeta
	if err := json.Unmarshal(v, &blockMeta); err != nil {
		return nil, fmt.Errorf("fail to unmarshal block meta from json: %w", err)
	}
	return &blockMeta, nil
}

// SaveBlockMeta persistent the given BlockMeta into storage
func (t *LevelDBBlockMetaAccessor) SaveBlockMeta(height int64, blockMeta *BlockMeta) error {
	key := t.getBlockMetaKey(height)
	buf, err := json.Marshal(blockMeta)
	if err != nil {
		return fmt.Errorf("fail to marshal block meta to json: %w", err)
	}
	return t.db.Put([]byte(key), buf, nil)
}

// GetBlockMetas returns all the block metas in storage , given chain client will prune it , so hopefully it won't be too much
func (t *LevelDBBlockMetaAccessor) GetBlockMetas() ([]*BlockMeta, error) {
	blockMetas := make([]*BlockMeta, 0)
	iterator := t.db.NewIterator(util.BytesPrefix([]byte(PrefixBlocMeta)), nil)
	defer iterator.Release()
	for iterator.Next() {
		buf := iterator.Value()
		if len(buf) == 0 {
			continue
		}
		var blockMeta BlockMeta
		if err := json.Unmarshal(buf, &blockMeta); err != nil {
			return nil, fmt.Errorf("fail to unmarshal block meta: %w", err)
		}
		blockMetas = append(blockMetas, &blockMeta)
	}
	return blockMetas, nil
}

// PruneBlockMeta remove all block meta that is older than the given block height
func (t *LevelDBBlockMetaAccessor) PruneBlockMeta(height int64) error {
	iterator := t.db.NewIterator(util.BytesPrefix([]byte(PrefixBlocMeta)), nil)
	defer iterator.Release()
	targetToDelete := make([][]byte, 0)
	for iterator.Next() {
		buf := iterator.Value()
		if len(buf) == 0 {
			continue
		}
		var blockMeta BlockMeta
		if err := json.Unmarshal(buf, &blockMeta); err != nil {
			return fmt.Errorf("fail to unmarshal block meta: %w", err)
		}
		if blockMeta.Height <= height {
			targetToDelete = append(targetToDelete, iterator.Key())
		}
	}
	for _, k := range targetToDelete {
		if err := t.db.Delete(k, nil); err != nil {
			return fmt.Errorf("fail to delete block meta with key(%s) from storage: %w", string(k), err)
		}
	}
	return nil
}

// UpsertTransactionFee update the transaction fee in storage
func (t *LevelDBBlockMetaAccessor) UpsertTransactionFee(fee float64, vSize int32) error {
	transactionFee := TransactionFee{
		Fee:   fee,
		VSize: vSize,
	}
	buf, err := json.Marshal(transactionFee)
	if err != nil {
		return fmt.Errorf("fail to marshal transaction fee struct to json: %w", err)
	}
	return t.db.Put([]byte(TransactionFeeKey), buf, nil)
}

// GetTransactionFee from db
func (t *LevelDBBlockMetaAccessor) GetTransactionFee() (float64, int32, error) {
	buf, err := t.db.Get([]byte(TransactionFeeKey), nil)
	if err != nil {
		return 0.0, 0, fmt.Errorf("fail to get transaction fee from storage: %w", err)
	}
	var transactionFee TransactionFee
	if err := json.Unmarshal(buf, &transactionFee); err != nil {
		return 0.0, 0, fmt.Errorf("fail to unmarshal transaction fee: %w", err)
	}
	return transactionFee.Fee, transactionFee.VSize, nil
}
