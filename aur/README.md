# AUR packaging

`PKGBUILD.in` is a release template rather than an installable PKGBUILD.

The tagged desktop release workflow builds the generic Linux bundle, calculates
its SHA-256, substitutes `@VERSION@` and `@SHA256@`, and publishes the final
checksum-pinned `PKGBUILD` as a GitHub release asset.

Copy that generated PKGBUILD into the AUR package repository when publishing a
desktop release. This keeps the AUR package pinned without checking `SKIP` into
the main Supernova repository.
