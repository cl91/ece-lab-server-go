//   Copyright 2009-2012 Joubin Houshyar
// 
//   Licensed under the Apache License, Version 2.0 (the "License");
//   you may not use this file except in compliance with the License.
//   You may obtain a copy of the License at
//    
//   http://www.apache.org/licenses/LICENSE-2.0
//    
//   Unless required by applicable law or agreed to in writing, software
//   distributed under the License is distributed on an "AS IS" BASIS,
//   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//   See the License for the specific language governing permissions and
//   limitations under the License.

// Package redis provides both clients and connectors for the Redis
// server.  Both synchronous and asynchronous interaction modes are
// supported.  Asynchronous clients (using the asynchronous connection)
// use pipelining.
//
// Synchronous semantics are defined by redis.Client interface type
//
//
// Usage example:
//
//  func usingRedisSync () Error {
//      spec := DefaultConnectionSpec();
//      pipeline := NewAsynchClient(spec);
//
//      value, reqErr := pipeline.Get("my-key");
//      if reqErr != nil { return withError (reqErr); }
//  }
//

package redis

import (
	"flag"
)

// The synchronous call semantics Client interface.
//
// Method names map one to one to the Redis command set.
// All methods may return an redis.Error, which is either
// a system error (runtime issue or bug) or Redis error (i.e. user error)
// See Error in this package for details of its interface.
//
// The synchronous client interface provides blocking call semantics supported by
// a distinct request/reply sequence at the connector level.
//
// Method names map one to one to the Redis command set.
//
// All methods may return an redis.Error, which is either a Redis error (from
// the server), or a system error indicating a runtime issue (or bug).
// See Error in this package for details of its interface.
type Client interface {

	// Redis QUIT command.
	Quit() (err Error)

	// Redis GET command.
	Get(key string) (result string, err Error)

	// Redis TYPE command.
	Type(key string) (result KeyType, err Error)

	// Redis SET command.
	Set(key string, arg1 string) Error

	// Redis SAVE command.
	Save() Error

	// Redis KEYS command using "*" wildcard 
	AllKeys() (result []string, err Error)

	// Redis KEYS command.
	Keys(key string) (result []string, err Error)

	// Redis EXISTS command.
	Exists(key string) (result bool, err Error)

	// Redis RENAME command.
	Rename(key, arg1 string) Error

	// Redis INFO command.
	Info() (result map[string]string, err Error)

	// Redis PING command.
	Ping() Error

	// Redis SETNX command.
	Setnx(key string, arg1 string) (result bool, err Error)

	// Redis GETSET command.
	Getset(key string, arg1 string) (result string, err Error)

	// Redis MGET command.
	Mget(key string, arg1 []string) (result []string, err Error)

	// Redis INCR command.
	Incr(key string) (result int64, err Error)

	// Redis INCRBY command.
	Incrby(key string, arg1 int64) (result int64, err Error)

	// Redis DECR command.
	Decr(key string) (result int64, err Error)

	// Redis DECRBY command.
	Decrby(key string, arg1 int64) (result int64, err Error)

	// Redis DEL command.
	Del(key string) (result bool, err Error)

	// Redis RANDOMKEY command.
	Randomkey() (result string, err Error)

	// Redis RENAMENX command.
	Renamenx(key string, arg1 string) (result bool, err Error)

	// Redis DBSIZE command.
	Dbsize() (result int64, err Error)

	// Redis EXPIRE command.
	Expire(key string, arg1 int64) (result bool, err Error)

	// Redis TTL command.
	Ttl(key string) (result int64, err Error)

	// Redis RPUSH command.
	Rpush(key string, arg1 string) Error

	// Redis LPUSH command.
	Lpush(key string, arg1 string) Error

	// Redis LSET command.
	Lset(key string, arg1 int64, arg2 string) Error

	// Redis LREM command.
	Lrem(key string, arg1 string, arg2 int64) (result int64, err Error)

	// Redis LLEN command.
	Llen(key string) (result int64, err Error)

	// Redis LRANGE command.
	Lrange(key string, arg1 int64, arg2 int64) (result []string, err Error)

	// Redis LTRIM command.
	Ltrim(key string, arg1 int64, arg2 int64) Error

	// Redis LINDEX command.
	Lindex(key string, arg1 int64) (result string, err Error)

	// Redis LPOP command.
	Lpop(key string) (result string, err Error)

	// Redis BLPOP command.
	Blpop(key string, timeout int) (result []string, err Error)

	// Redis RPOP command.
	Rpop(key string) (result string, err Error)

	// Redis BRPOP command.
	Brpop(key string, timeout int) (result []string, err Error)

	// Redis RPOPLPUSH command.
	Rpoplpush(key string, arg1 string) (result string, err Error)

	// Redis BRPOPLPUSH command.
	Brpoplpush(key string, arg1 string, timeout int) (result []string, err Error)

	// Redis SADD command.
	Sadd(key string, arg1 string) (result bool, err Error)

	// Redis SREM command.
	Srem(key string, arg1 string) (result bool, err Error)

	// Redis SISMEMBER command.
	Sismember(key string, arg1 string) (result bool, err Error)

	// Redis SMOVE command.
	Smove(key string, arg1 string, arg2 string) (result bool, err Error)

	// Redis SCARD command.
	Scard(key string) (result int64, err Error)

	// Redis SINTER command.
	Sinter(key string, arg1 []string) (result []string, err Error)

	// Redis SINTERSTORE command.
	Sinterstore(key string, arg1 []string) Error

	// Redis SUNION command.
	Sunion(key string, arg1 []string) (result []string, err Error)

	// Redis SUNIONSTORE command.
	Sunionstore(key string, arg1 []string) Error

	// Redis SDIFF command.
	Sdiff(key string, arg1 []string) (result []string, err Error)

	// Redis SDIFFSTORE command.
	Sdiffstore(key string, arg1 []string) Error

	// Redis SMEMBERS command.
	Smembers(key string) (result []string, err Error)

	// Redis SRANDMEMBER command.
	Srandmember(key string) (result string, err Error)

	// Redis ZADD command.
	Zadd(key string, arg1 float64, arg2 string) (result bool, err Error)

	// Redis ZREM command.
	Zrem(key string, arg1 string) (result bool, err Error)

	// Redis ZCARD command.
	Zcard(key string) (result int64, err Error)

	// Redis ZSCORE command.
	Zscore(key string, arg1 string) (result float64, err Error)

	// Redis ZRANGE command.
	Zrange(key string, arg1 int64, arg2 int64) (result []string, err Error)

	// Redis ZREVRANGE command.
	Zrevrange(key string, arg1 int64, arg2 int64) (result []string, err Error)

	// Redis ZRANGEBYSCORE command.
	Zrangebyscore(key string, arg1 float64, arg2 float64) (result []string, err Error)

	// Redis HGET command.
	Hget(key string, hashkey string) (result string, err Error)

	// Redis HSET command.
	Hset(key string, hashkey string, arg1 string) Error

	// Redis HGETALL command.
	Hgetall(key string) (result []string, err Error)

	// Redis FLUSHDB command.
	Flushdb() Error

	// Redis FLUSHALL command.
	Flushall() Error

	// Redis MOVE command.
	Move(key string, arg1 int64) (result bool, err Error)

	// Redis BGSAVE command.
	Bgsave() Error

	// Redis LASTSAVE command.
	Lastsave() (result int64, err Error)

	// Redis PUBLISH command.
	// Publishes a message to the named channels.  This is a blocking call.
	//
	// Returns the number of PubSub subscribers that received the message.
	// OR error if any.
	Publish(channel string, message string) (recieverCout int64, err Error)
}

// ----------------
// flags
//
// go-redis will make use of command line flags where available.  flag names
// for this package are all prefixed by "redis:" to prevent possible name collisions.
//
// Note that because flag.Parse() can only be called once, add all flags must have
// been defined by the time it is called, we CAN NOT call flag.Parse() in our init()
// function, as that will prevent any invokers from defining their own flags.
//
// It is your responsibility to call flag.Parse() at the start of your main().

// redis:d
//
// global debug flag for redis package components.
// 
var _debug *bool = flag.Bool("redis:d", false, "debug flag for go-redis") // TEMP: should default to false
func debug() bool {
	return *_debug
}
