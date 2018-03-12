# Built-in Packs

This directory contains the built-in Draft packs which are installed via `draft init`.

_If you are interested in creating your own packs_, you can simply create those packs in your local `$(draft home)/packs` directory.

```
packs/github.com/Azure/draft/packs
  |
  |- PACKNAME
  |     |
  |     |- charts/
  |     |    |- Chart.yaml
  |     |    |- ...
  |     |- Dockerfile
  |     |- detect
  |     |- ...
  |
  |- PACK2
        |-...
```
