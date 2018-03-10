gog-backup: Backups for your GoG.com games
==========================================

**GitHub:** https://github.com/mscharley/gog-backup  
**Author:** Matthew Scharley  
**Contributors:** [See contributors on GitHub][gh-contrib]  
**Bugs/Support:** [Github Issues][gh-issues]  
**Copyright:** 2018  
**License:** [MIT license][license]

Synopsis
--------

Backups for games and other media attached to your GoG.com account.

Installation
------------

```
go get github.com/mscharley/gog-backup/cmd/gog-backup
```

Usage
-----

```
gog-backup -help
```

Configuration
-------------

Any command-line parameters can be placed in an ini file anywhere you like (I use `~/.gog-backup.ini`) and then
passed in using `gog-backup -config ~/.gog-backup.ini`.

  [license]: https://raw.github.com/mscharley/gog-backup/master/LICENSE
  [gh-contrib]: https://github.com/mscharley/gog-backup/graphs/contributors
  [gh-issues]: https://github.com/mscharley/gog-backup/issues
