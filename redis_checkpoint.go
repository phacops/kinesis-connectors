package connector

import (
	"fmt"
	"sync"

	"github.com/apex/log"
	"github.com/go-redis/redis"
)

// RedisCheckpoint implements the Checkpont interface.
// Used to enable the Pipeline.ProcessShard to checkpoint it's progress
// while reading records from Kinesis stream.
type RedisCheckpoint struct {
	AppName    string
	StreamName string

	client *redis.Client

	sequenceMutex   sync.RWMutex
	sequenceNumbers map[string]string
}

// CheckpointExists determines if a checkpoint for a particular Shard exists.
// Typically used to determine whether we should start processing the shard with
// TRIM_HORIZON or AFTER_SEQUENCE_NUMBER (if checkpoint exists).
func (c *RedisCheckpoint) CheckpointExists(shardID string) bool {
	val, _ := c.client.Get(c.key(shardID)).Result()

	if val != "" {
		c.sequenceMutex.Lock()
		c.sequenceNumbers[shardID] = val
		c.sequenceMutex.Unlock()

		return true
	}

	return false
}

// SequenceNumber returns the current checkpoint stored for the specified shard.
func (c *RedisCheckpoint) SequenceNumber(shardID string) string {
	c.sequenceMutex.RLock()
	defer c.sequenceMutex.RUnlock()

	return c.sequenceNumbers[shardID]
}

// SetCheckpoint stores a checkpoint for a shard (e.g. sequence number of last record processed by application).
// Upon failover, record processing is resumed from this point.
func (c *RedisCheckpoint) SetCheckpoint(shardID string, sequenceNumber string) {
	err := c.client.Set(c.key(shardID), sequenceNumber, 0).Err()

	if err != nil {
		log.WithError(err).Error("redis checkpoint set")
	}

	c.sequenceMutex.Lock()
	c.sequenceNumbers[shardID] = sequenceNumber
	c.sequenceMutex.Unlock()
}

// key generates a unique Redis key for storage of Checkpoint.
func (c *RedisCheckpoint) key(shardID string) string {
	return fmt.Sprintf("%v:checkpoint:%v:%v", c.AppName, c.StreamName, shardID)
}
