# Watch: continuously iterate a queue until it is empty

## Arguments

`$ARGUMENTS` is the queue number to watch: `0`, `1`, or `2`.

## Instructions

Run the following to continuously process the queue every 30 seconds:

```
/loop 60s /iterate $ARGUMENTS
```
