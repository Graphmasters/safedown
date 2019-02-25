# Safedown

Safedown is for ensuring that an application shuts down gracefully even when termination or interruption signals are sent. 
The following snippet shows how to ensure text is printed when the application ends naturally or or is terminated. 

```go
sa := NewShutdownActions(FirstInLastDone, syscall.SIGTERM, syscall.SIGINT)
defer sa.Shutdown()

sa.AddActions(func() {
	fmt.Println("... and this will be done last.")
})

sa.AddActions(func() {
	fmt.Println("This will be done first ...")
})
```