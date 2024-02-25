# Initial-Go-Server
me walking through the web server tutorial on boot.dev (https://www.boot.dev/assignments/861ada77-c583-42c8-a265-657f2c453103)

## What is this?
this project serves as a basic "clone" of Twitter, meant as a jumping off point for learning how to build a RESTful API

## Why is this here?
I'll probably use this in the future as a reference for things like:

- Serving HTML pages
- Parsing and responding with JSON
- Authenticating user requests
- Responding to webhook events
- Seperating concerns

## How do I download and use this project?
I don't see why you'd want to, but

first you'd need to clone the repo on to your machine with `git clone`

then you'd need to add a .env folder with the following keys:

```
JWT_SECRET=<some string you generate>
POLKA_KEY=f271c81ff7084ee5b99a5091b42d486e
```
then all you need to do is run

```bash
$ go build
$ go-chirpy
```
you can also run `go-chirpy` with an optional `--debug` flag to delete the stored database before spinning up the server

