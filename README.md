# Distributed Testing Framework for Rust

![](https://i.imgur.com/ncXtxDp.png)

## Motivation

The idea for this project was born when I was working on [Rush - a shell, written in Rust](https://github.com/luchev/rush). Because Rust runs tests in parallel and a shell's tests change the environment variables, a lot of my unit tests would randomly fail because they are not ran in isolation. Hence the idea to build a framework, which runs tests in isolated environments and scales horizontally was born.

## Requirements

This project works only on Linux!

In order to run the app you will need [Go](https://golang.org/) and [Rust](https://www.rust-lang.org/).

Also, the Rust setup requires the [cargo-junit](https://github.com/luchev/cargo-junit). This is a fork of a cargo extension, which convertes cargo output to JUnit format for easier parsing. Sadly `cargo-junit` depends on `cargo-results`, which was broken for recent (2021) Rust versions, therefore you need to have this [cargo-results fork](https://github.com/luchev/cargo-results) for the app to work. Hopefully we will see better tooling for Rust tests soon, but until then, this project will rely on these 2 forks.

## Get Started

To get setup, first clone the repo and enter the `src` directory.

```
git clone https://github.com/luchev/distributed-testing-framework-for-rust

cd distributed-testing-framework-for-rust/src
```

Next you must start a Master service first on port 8080 (the app won't work if your Master service is on a different port)

```
go run server.go --master --port 8080
```

Now you can go to [http://127.0.0.1:8080](http://127.0.0.1:8080) and try out the app.

Alternatively you can also spin up a few Workers. You can start them on any port you like, in the example below I've chosen port 8081. After the Worker is started you have to let the Master know about the Worker's existence. You can register a new Worker here [http://127.0.0.1:8080/add_node](http://127.0.0.1:8080/add_node)

```
go run server.go --worker --port 8081
```

You're all set!

## Issues and Possible improvements

Known issues:

- Very long build times because ... this is Rust
- If the Master node dies, the system becomes unoperational

Possible Improvements:

- In order to reduce build times a build cache could be used
- Dynamically assign a Worker to become the Master if the Master goes down

## More examples

Here's what the Worker status page looks like

![](https://i.imgur.com/kwfiM5Z.png)

And this is all of Rush's tests running smoothly and passing successfully :)

![](https://i.imgur.com/kRAIVtA.png)

Finally, a preview of the logs of 1 Master and 2 Worker nodes being started

![](https://i.imgur.com/FBxyOaY.png)
