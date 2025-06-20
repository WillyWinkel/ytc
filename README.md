# ytc

A web application for Yang Tai Chi Hamburg, featuring calendar, news, downloads, and more.

## Features

- Multi-language support (German, English)
- Embedded static assets (images, downloads, templates)
- Calendar and news integration via iCal/webcal
- Download page with file descriptions
- Responsive UI with Bootstrap

## Dependencies

- [Go 1.20+](https://golang.org/dl/)
- [github.com/arran4/golang-ical](https://github.com/arran4/golang-ical)

Install Go dependencies:

```sh
go mod tidy
```

## Developer Commands

### Build

Build the server binary (with embedded static files):

```sh
go build -o ytc-server .
```

### Run

Run the server locally:

```sh
./ytc-server
```

The server will start at [http://localhost:8080](http://localhost:8080).

### Test

Run all tests with verbose output:

```sh
go test -v ./...
```

### Clean

Remove the built binary:

```sh
rm -f ytc-server
```

## Project Structure

- `internal/app/` - Main application code
- `static/` - Static assets (images, downloads, templates)
- `.github/workflows/` - CI/CD GitHub Actions

## CI/CD

On every push to `main`, GitHub Actions will:

- Build the project
- Run all tests
- Upload the built binary as a downloadable artifact

See `.github/workflows/build.yml` for details.

## License

MIT License (see LICENSE file)
