# Draft Pack Repository Plugin

The Draft pack repository plugin.

## Installation

Fetch the latest version of `draft pack-repo` using

```
$ draft plugin install https://github.com/Azure/draft-pack-repo
```

## Why a Plugin?

Draft Pack Repository Plugin (or `draft pack-repo` for short) enables users to fetch, list, add and
remove pack repositories to bootstrap all of their internal and external projects. It is incredibly
opinionated on how to fetch, list, add and remove these repositories whereas Draft core does not
care about these concepts.

This also enables the Draft community to come up alternative forms of pack repositories by
implementing their own plugin for fetching down these packs, so it made sense to initially spike the
tooling as an entirely separate project.

## Contributing

This project welcomes contributions and suggestions.  Most contributions require you to agree to a
Contributor License Agreement (CLA) declaring that you have the right to, and actually do, grant us
the rights to use your contribution. For details, visit https://cla.microsoft.com.

When you submit a pull request, a CLA-bot will automatically determine whether you need to provide
a CLA and decorate the PR appropriately (e.g., label, comment). Simply follow the instructions
provided by the bot. You will only need to do this once across all repos using our CLA.

This project has adopted the [Microsoft Open Source Code of Conduct](https://opensource.microsoft.com/codeofconduct/).
For more information see the [Code of Conduct FAQ](https://opensource.microsoft.com/codeofconduct/faq/) or
contact [opencode@microsoft.com](mailto:opencode@microsoft.com) with any additional questions or comments.
