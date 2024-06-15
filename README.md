![cd](cdtools.png) 
# Cloud Dragon  (WIP)
A swiss-knife synchronisation server that will make your life easier. Featuring:

* **Distributed Locks**
    * Long-Polling - Lock for one client returns right after unlock made by another
    * Persistent - Locks continue to block upon reboot and can be unlocked
    * Minimal latency & performance overhead  (~1-5ms)
    * Up to 100 sequential lock/unlock operations per second for a single key
    * Up to 10k clients waiting on a lock for a single key
* **Key-Value operations**
    * Strong consistency guarantees - all operations either succeed or fail.
    * Long-Polling notifications - http watch request unblocks after kv value was changed
    * Up to 100k req/sec on an average server
    * Up to 10k watchers for a single key
* **Atomic operations**
    * Counters
    * Sequences
    * CompareAndSwap
    * Up to 50k updates/sec for a single counter
* **Backups**
    * Automatic backups to S3
    * Import / Export of data.

Spend $5 on single server and save months of your time on not worrying about:
- database transactions & conflicts
- concurrent background actions executed at the same time
- on-the-fly config updates
- redis setup & maintenance just to cache a few values

## Use-cases
* Ensure exclusive code execution with minimal latency overhead.
* Generate short, sequential IDs
* Share configuration among multiple servers that is updated instantly
* Cache values in centralized storage


## API Design:
API is divided into "accounts" (aka partitions), where 1 account can't share any data with another account. 

If you never expect to have more than 50k/req second for all your use-cases - you can only 1 account and that's it. If you may need more performance in future - you can create multiple accounts (aka tenants). This will allow you to put Load Balancer in front of this service and distribute load across multiple machines, based on account in http url path.

All actions are executed atomically. If you issue a Lock, update 100 keys & increment 50 counters in a single request - all those actions will either succeed or fail together.

## Usage Examples:
Lock key "ABC" for 30 seconds. Wait for 30 seconds to acquire the lock
```
POST /db/my_env
{
    "LockID": "ABC",
    "LockDur": 30,
    "LockWait": 30
}
resp 200:
{
    "Lock":  1235553,
    "Till": 172343434, // Unix
}
```

Set some values & increment counter
```
POST /db/my_env
{
    "KVSet": ["Key": "ABC", "Value": "123"],
    "KVGet":  ["Key": "CDE"],
    "Atomic": ["Key": "Total_Count", "Add": 1]
}
resp 200:
{
    "Atomic": ["Key": "Total_Count", "Value": 333]
    "KV": ["Key": "CDE", "Value": { "my_json": "object" }]
}
```

Unlock id
```
POST /db/my_env
{
    "UnlockID": "ABC",
    "Unlock": "1235553",
}
resp 200:
{}
```


Idempotent update (without lock)
```
POST /db/my_env
{
    "IdempotencyIDs": ["ABC_create"],
    "KVSet": ["Key": "ABC", "Value": "123"],
    "Atomic": ["Key": "Total_Count", "Add": 1]
}
resp 200:
{
    "Atomic": ["Key": "Total_Count", "Value": 333]
}
```

Try duplicate request
```
POST /db/my_env
{
    "IdempotencyIDs": ["ABC_create"],
    "KVSet": ["Key": "ABC", "Value": "123"],
    "Atomic": ["Key": "Total_Count", "Add": 1]
}
resp 409:
{
    "Code": "duplicate_request"
}
```


Watch for key change
```
POST /watch/my_env
{
    "ID": "ABC",
    "Version": 0, // watch for key creation
}
resp - ... blocking until the key is created
```

Update key from another app
```
POST /db/my_env
{
    "KVSet": ["Key": "ABC", "Value": "123"],
}
```

Now watch request unblocks
```
... resp
{
    "Key": "ABC",
    "Version": 54,
    "Value": "123"
}
```

Try to wait for key change, but key was already changed since we looked at it last time
```
POST /watch/my_env
{
    "ID": "ABC",
    "Version": 32, // last version we had was 32
}
resp - unblocked immediately - current version is higher
{
    "Key": "ABC",
    "Version": 54,
    "Value": "123"
}
```

Try to wait for key change, but for current version of the key
```
POST /watch/my_env
{
    "ID": "ABC",
    "Version": 54, // watch for key change
}
resp ... - blocked until value is updated again
{
    "Key": "ABC",
    "Version": 54,
    "Value": "123"
}
```



## Benchmarks
GOMAXPROCS=4 (2 cores) on AMD Ryzen 5 6600H.  (The rest of the cores is used for running benchmark)
 ```
2024/06/15 17:07:33 WatchReaction 1 key 1 watcher: 5 ms 
2024/06/15 17:07:33 WatchReaction 1 key 100 watchers: 8 ms 
2024/06/15 17:07:34 WatchReaction 1 key 1000 watchers: 16 ms 
2024/06/15 17:07:34 WatchReaction 1 key 10000 watchers: 11 ms 
2024/06/15 17:07:34 WatchReaction 1000 keys 5 watchers per key: min 3 ms,avg 29.9 ms,  max 63 ms,  total delay: 370 ms 
2024/06/15 17:07:35 LockUnlock for 1 account and 1 key (sequential): 0.2k req/sec 
2024/06/15 17:07:37 LockUnlock for 1 account and 1000 keys: 53.3k req/sec 
2024/06/15 17:07:38 LockUnlock for 1000 accounts and 1000 keys: 68.1k req/sec 
2024/06/15 17:07:42 64byte write KV for 1 account and 1 key: 20.8k req/sec 1.3 MB/sec
2024/06/15 17:07:45 64byte write KV for 1 account and 1000 keys: 28.3k req/sec 1.8 MB/sec
2024/06/15 17:07:46 64byte write KV for 1000 accounts and 1 keys: 92.5k req/sec 5.9 MB/sec
2024/06/15 17:07:47 64byte write KV for 1000 accounts and 1000 keys: 107.7k req/sec 6.9 MB/sec
2024/06/15 17:07:51 1KB write KV for 1 account and 1 key: 22.2k req/sec 22.8 MB/sec
2024/06/15 17:07:55 1KB write KV for 1 account and 1000 keys: 26.8k req/sec 27.4 MB/sec
2024/06/15 17:07:56 1KB write KV for 1000 accounts and 1 keys: 63.0k req/sec 64.6 MB/sec
2024/06/15 17:07:59 1KB write KV for 1000 accounts and 1000 keys: 40.9k req/sec 41.9 MB/sec
2024/06/15 17:08:00 10 KB write KV for 1 account and 1 key: 2.7k req/sec 27.3 MB/sec
2024/06/15 17:08:01 10 KB write KV for 1 account and 1000 keys: 10.5k req/sec 107.1 MB/sec
2024/06/15 17:08:02 10 KB write KV for 1000 accounts and 1 keys: 7.1k req/sec 72.4 MB/sec
2024/06/15 17:08:04 10 KB write KV for 1000 accounts and 1000 keys: 4.7k req/sec 47.7 MB/sec
2024/06/15 17:08:05 atomic for 1 account and 1 key: 32.9k req/sec
2024/06/15 17:08:05 atomic for 1 account and 1000 keys: 18.4k req/sec
2024/06/15 17:08:05 atomic for 1000 accounts and 1 keys: 88.7k req/sec
2024/06/15 17:08:06 atomic for 1000 accounts and 1000 keys: 36.2k req/sec

```

