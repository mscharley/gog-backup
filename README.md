# gog-backup: Backups for your GoG.com games

**GitHub:** https://github.com/mscharley/gog-backup  
**Author:** Matthew Scharley  
**Contributors:** [See contributors on GitHub][gh-contrib]  
**Bugs/Support:** [Github Issues][gh-issues]  
**Copyright:** 2021  
**License:** [MIT license][license]

## Synopsis

Backups for games and other media attached to your GoG.com account.

## Installation

```console
go get github.com/mscharley/gog-backup/cmd/gog-backup
```

## Docker

```console
docker run -t -v gog-backup.ini:/etc/gog-backup.ini ghcr.io/mscharley/gog-backup
```

## Usage

```console
gog-backup -help
```

## Configuration

[You will need access to a refresh token as described here.][auth-docs]

You may place any command-line parameters in an ini file anywhere you like (I use `~/.gog-backup.ini`) and then
passed in using `gog-backup -config ~/.gog-backup.ini`.

```ini
refresh-token = "foobar"
```

[license]: https://raw.github.com/mscharley/gog-backup/master/LICENSE
[gh-contrib]: https://github.com/mscharley/gog-backup/graphs/contributors
[gh-issues]: https://github.com/mscharley/gog-backup/issues
[auth-docs]: https://gogapidocs.readthedocs.io/en/latest/auth.html
