package redis

import "github.com/redis/go-redis/v9"

// releaseLockScript atomically checks that the lock is still owned by a given instance before deleting it.
// This prevents a worker from accidentally releasing a lock it no longer owns due to TTL expiry.
//
// KEYS[1] - the lock key
// ARGV[1] - the instance ID (lock token)
//
// Returns 1 if the lock was released, 0 if the lock was not owned by this instance.
var releaseLockScript = redis.NewScript(`
if redis.call("GET", KEYS[1]) == ARGV[1] then
    return redis.call("DEL", KEYS[1])
else
    return 0
end
`)
