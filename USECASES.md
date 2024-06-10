#### State Machine

```
Req
{
    "lock": {
        id: "123",
        wait: 10,
        duration: 30,
    },
    get: ["123"]
}
Resp
{
    "lock": {
        "status": "locked|not_locked|recover(not properly unlocked previously)",
        "handle": "123123",
        "till": "17543343444"
    }
    "get": {"123": ...state... }
}
```

Update state in code, handle recover case, then run
```
Req
{
    "unlock": "123123",
    "set": {
        "123": ...newstate...
    }
}
Resp
{}
```

#### State Machine (External DB)

```
Req
{
    "lock": {
        id: "123",
        wait: 10,
        duration: 30,
    },
}
Resp
{
    "lock": {
        "handle": "123123",
        "till": "17543343444"
    }
}
```

Get from external DB, update state in code, handle recover case, update in external DB, then run
```
Req
{
    "unlock": "123123",
}
Resp
{}
```
