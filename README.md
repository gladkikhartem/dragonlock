![cd](cdtools.png) 
# Cloud Dragon  (WIP)
A swiss-knife synchronisation server that will make your life easier. Featuring:

* **Distributed Locks**
    * Long-Polling - Lock for one client returns right after unlock made by another
    * Persistent - Locks continue to block upon reboot and can be unlocked
    * Minimal latency & performance overhead  (~1-5ms)
    * Up to 1000 lock/unlock operations per second for a single lock
    * Up to 10k clients performing lock on the same key
* **Key-Value storage**
    * Strong consistency guarantees and versioning
    * Long-Polling notifications - http watch request unblocks after kv value was changed
    * Up to 100k req/sec on an average server
* **Atomic operations**
    * Persistent - all changes are flushed to disk
    * Up to 100k req/sec for a single counter on an average server

Spend $5 on single server and save months of your time on not worrying about:
- database transactions & conflicts
- concurrent background actions executed at the same time
- on-the-fly config updates
- redis maintenance & scaling just to store few values

## Use-cases
* Ensure exclusive code execution with minimal latency overhead.
* Generate short, sequential IDs
* Share configuration among multiple servers that is updated instantly
* Cache values in centralized storage


## API / Examples:
Lock key "ABC" for 30 seconds. Wait for 30 seconds to acquire the lock
```
POST /db/dev
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
POST /db/dev
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
POST /db/dev
{
    "UnlockID": "ABC",
    "Unlock": "1235553",
}
resp 200:
{}
```


Idempotent update (without lock)
```
POST /db/dev
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
POST /db/dev
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


## Benchmarks
GOMAXPROCS=4 (2 cores) on AMD Ryzen 5 6600H.  (The rest of the cores is used for running benchmark)
 ```
2024/06/15 17:00:44 WatchReaction 1 key 1 watcher: 9 ms 
2024/06/15 17:00:44 WatchReaction 1 key 100 watchers: 9 ms 
2024/06/15 17:00:44 WatchReaction 1 key 100 watchers: 23 ms 
2024/06/15 17:00:45 WatchReaction 1 key 10000 watchers: 19 ms 
2024/06/15 17:00:45 WatchReaction 1000 keys 5 watchers per key: min 9 ms,avg 54.7 ms,  max 81 ms,  total delay: 383 ms 
2024/06/15 17:00:48 LockUnlock for 1 account and 1000 keys: 35.0k req/sec 
2024/06/15 17:00:49 LockUnlock for 1000 accounts and 1000 keys: 65.5k req/sec 
2024/06/15 17:00:53 64byte write KV for 1 account and 1 key: 20.2k req/sec 1.3 MB/sec
2024/06/15 17:00:56 64byte write KV for 1 account and 1000 keys: 27.9k req/sec 1.8 MB/sec
2024/06/15 17:00:57 64byte write KV for 1000 accounts and 1 keys: 124.7k req/sec 8.0 MB/sec
2024/06/15 17:00:58 64byte write KV for 1000 accounts and 1000 keys: 121.5k req/sec 7.8 MB/sec
2024/06/15 17:01:01 1KB write KV for 1 account and 1 key: 23.3k req/sec 23.9 MB/sec
2024/06/15 17:01:05 1KB write KV for 1 account and 1000 keys: 26.7k req/sec 27.3 MB/sec
2024/06/15 17:01:07 1KB write KV for 1000 accounts and 1 keys: 71.4k req/sec 73.1 MB/sec
2024/06/15 17:01:08 1KB write KV for 1000 accounts and 1000 keys: 54.7k req/sec 56.0 MB/sec
2024/06/15 17:01:09 10 KB write KV for 1 account and 1 key: 3.7k req/sec 38.2 MB/sec
2024/06/15 17:01:10 10 KB write KV for 1 account and 1000 keys: 10.0k req/sec 102.0 MB/sec
2024/06/15 17:01:12 10 KB write KV for 1000 accounts and 1 keys: 7.4k req/sec 75.8 MB/sec
2024/06/15 17:01:13 10 KB write KV for 1000 accounts and 1000 keys: 6.4k req/sec 65.0 MB/sec
2024/06/15 17:01:13 atomic for 1 account and 1 key: 34.9k req/sec
2024/06/15 17:01:14 atomic for 1 account and 1000 keys: 16.8k req/sec
2024/06/15 17:01:14 atomic for 1000 accounts and 1 keys: 73.5k req/sec
2024/06/15 17:01:14 atomic for 1000 accounts and 1000 keys: 38.6k req/sec
```