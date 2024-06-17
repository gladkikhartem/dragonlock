![cd](cdtools.png) 
# Dragon Lock  (WIP)
A swiss-knife synchronisation server that will make your life easier. Featuring:

* **Distributed Locks**
    * Long-Polling - Lock for one client returns right after unlock made by another
    * Persistent - Locks continue to block upon reboot and can be unlocked
    * Minimal latency & performance overhead  (~1-5ms)
    * Up to 1k sequential lock/unlock operations per second for a single key
    * Up to 10k clients waiting on a lock for a single key
* **Storing shared data**
    * Strong consistency guarantees - all operations either succeed or fail.
    * Long-Polling notifications - http watch request unblocks after kv value was changed
    * Up to 100k req/sec on an average server
    * Up to 10k watchers for a single key
* **Atomic operations**
    * Counters, Sequences, CompareAndSwap
    * Idempotency checks included
    * All updates persisted to disk
    * Up to 50k updates/sec for a single counter

Spend $5 on single server and save months of your time on not worrying about:
- database transactions & conflicts
- concurrent background actions executed at the same time
- on-the-fly config updates
- redis setup & maintenance just to share a few values between apps

## Current status
This is Work In Progress right now.
Stability testing, docs, admin endpoints & UI are required.

## API Guarantees:
Whole request is executed atomically - either all changes applied or none.

All changes are persisted to disk before success response is returned.

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
- GOMAXPROCS=4 on AMD Ryzen 5 6600H (2 CPU cores for API, 4 CPU cores for benchmark client).
- 400 clients
- Simulated Network Latency:  ~1ms
 ```
2024/06/17 11:49:49 WatchReaction 1 key 1 watcher: 7 ms 
2024/06/17 11:49:49 WatchReaction 1 key 100 watchers: 8 ms 
2024/06/17 11:49:50 WatchReaction 1 key 1000 watchers: 36 ms 
2024/06/17 11:49:50 WatchReaction 1 key 10000 watchers: 29 ms 
2024/06/17 11:49:50 WatchReaction 1000 keys 5 watchers per key: min 6 ms,avg 62.9 ms,  max 94 ms,  total delay: 395 ms 
2024/06/17 11:49:50 LockUnlock for 1 account and 1 key (sequential): 866.6 req/sec 
2024/06/17 11:49:52 LockUnlock for 1 account and 1000 keys: 87.5k req/sec 
2024/06/17 11:49:53 LockUnlock for 1000 accounts and 1000 keys: 96.9k req/sec 
2024/06/17 11:49:55 64byte write KV for 1 account and 1 key: 25.3k req/sec 1.6 MB/sec
2024/06/17 11:49:59 64byte write KV for 1 account and 1000 keys: 27.5k req/sec 1.8 MB/sec
2024/06/17 11:50:00 64byte write KV for 1000 accounts and 1 keys: 105.1k req/sec 6.7 MB/sec
2024/06/17 11:50:01 64byte write KV for 1000 accounts and 1000 keys: 105.3k req/sec 6.7 MB/sec
2024/06/17 11:50:04 1KB write KV for 1 account and 1 key: 23.0k req/sec 23.5 MB/sec
2024/06/17 11:50:08 1KB write KV for 1 account and 1000 keys: 28.1k req/sec 28.8 MB/sec
2024/06/17 11:50:09 1KB write KV for 1000 accounts and 1 keys: 68.4k req/sec 70.1 MB/sec
2024/06/17 11:50:11 1KB write KV for 1000 accounts and 1000 keys: 59.2k req/sec 60.7 MB/sec
2024/06/17 11:50:12 10 KB write KV for 1 account and 1 key: 3.9k req/sec 39.7 MB/sec
2024/06/17 11:50:13 10 KB write KV for 1 account and 1000 keys: 10.3k req/sec 105.3 MB/sec
2024/06/17 11:50:14 10 KB write KV for 1000 accounts and 1 keys: 8.4k req/sec 85.6 MB/sec
2024/06/17 11:50:16 10 KB write KV for 1000 accounts and 1000 keys: 6.5k req/sec 66.2 MB/sec
2024/06/17 11:50:16 atomic for 1 account and 1 key: 31.8k req/sec
2024/06/17 11:50:17 atomic for 1 account and 1000 keys: 19.9k req/sec
2024/06/17 11:50:17 atomic for 1000 accounts and 1 keys: 60.3k req/sec
2024/06/17 11:50:17 atomic for 1000 accounts and 1000 keys: 30.1k req/sec
```

- GOMAXPROCS=4 on AMD Ryzen 5 6600H (2 CPU cores for API, 4 CPU cores for benchmark client).
- 400 clients
- Simulated Network Latency:  ~10ms
```
2024/06/17 11:52:10 WatchReaction 1 key 1 watcher: 17 ms 
2024/06/17 11:52:11 WatchReaction 1 key 100 watchers: 20 ms 
2024/06/17 11:52:11 WatchReaction 1 key 1000 watchers: 40 ms 
2024/06/17 11:52:11 WatchReaction 1 key 10000 watchers: 26 ms 
2024/06/17 11:52:12 WatchReaction 1000 keys 5 watchers per key: min 22 ms,avg 73.0 ms,  max 110 ms,  total delay: 411 ms 
2024/06/17 11:52:12 LockUnlock for 1 account and 1 key (sequential): 186.2 req/sec 
2024/06/17 11:52:16 LockUnlock for 1 account and 1000 keys: 29.6k req/sec 
2024/06/17 11:52:18 LockUnlock for 1000 accounts and 1000 keys: 41.3k req/sec 
2024/06/17 11:52:20 64byte write KV for 1 account and 1 key: 22.8k req/sec 1.5 MB/sec
2024/06/17 11:52:22 64byte write KV for 1 account and 1000 keys: 56.5k req/sec 3.6 MB/sec
2024/06/17 11:52:23 64byte write KV for 1000 accounts and 1 keys: 59.4k req/sec 3.8 MB/sec
2024/06/17 11:52:25 64byte write KV for 1000 accounts and 1000 keys: 61.0k req/sec 3.9 MB/sec
2024/06/17 11:52:27 1KB write KV for 1 account and 1 key: 25.7k req/sec 26.3 MB/sec
2024/06/17 11:52:30 1KB write KV for 1 account and 1000 keys: 44.5k req/sec 45.6 MB/sec
2024/06/17 11:52:32 1KB write KV for 1000 accounts and 1 keys: 48.4k req/sec 49.6 MB/sec
2024/06/17 11:52:34 1KB write KV for 1000 accounts and 1000 keys: 47.6k req/sec 48.8 MB/sec
2024/06/17 11:52:35 10 KB write KV for 1 account and 1 key: 3.4k req/sec 34.7 MB/sec
2024/06/17 11:52:36 10 KB write KV for 1 account and 1000 keys: 9.8k req/sec 100.9 MB/sec
2024/06/17 11:52:37 10 KB write KV for 1000 accounts and 1 keys: 8.5k req/sec 87.2 MB/sec
2024/06/17 11:52:38 10 KB write KV for 1000 accounts and 1000 keys: 6.3k req/sec 64.3 MB/sec
2024/06/17 11:52:39 atomic for 1 account and 1 key: 41.6k req/sec
2024/06/17 11:52:39 atomic for 1 account and 1000 keys: 20.0k req/sec
2024/06/17 11:52:39 atomic for 1000 accounts and 1 keys: 52.0k req/sec
2024/06/17 11:52:40 atomic for 1000 accounts and 1000 keys: 31.6k req/sec
```

