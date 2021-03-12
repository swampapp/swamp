# Event Bus

Swamp uses a lightweight wrapper for [a lightweight event bus](https://github.com/mustafaturan/bus).

Each UI component may emit its own events and listen to other component events.

The events are processed synchronously (bus module design choice), so handlers processing events that may take a while should use goroutines or it'll block the UI.

I may enhance bus to handle events asynchronously eventually, but synchronous processing + goroutines works for now.

## Singleton Bus

There's only one global bus to handler events currently. To use it, simply import the `eventbus` internal package and use the public methods.

## Convention for registering and emitting events from a component

* New events are declared inside the package as public package constants, adding the `Event` suffix to the constant name. The value should be the package name, a dot, and the event name and use underscore to separate words:

```go
package packagename

const MyAwesomeEvent = "packagename.something_happened"
```

* The package should register the events it's responsible for, so it can emit them:

```go

func New() *MyAwesomeComponent {
  eventbus.RegisterEvents(MyAwesomeEvent)
}
```

Make sure you have register the events the package will emit before emitting them, or you'll get an error.

* Emitting events:

```go
func beAwesome() {
	eventbus.Emit(context.Background(), MyAwesomeEvent, "sample-payload")
}
```