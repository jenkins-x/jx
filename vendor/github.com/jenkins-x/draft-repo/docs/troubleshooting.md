## Troubleshooting

Common issues or questions that users have run into when using Draft are detailed below.

### My repository is detected as the wrong language

`draft create` displays languages percentages for the files in the repository. The percentages are calculated based on the bytes of code for each language as reported by [pkg/linguist](https://github.com/Azure/draft/tree/master/pkg/linguist), which is a Go port of [github linguist](https://github.com/github/linguist). If `draft create` is reporting a language that you don't expect:

1. Use `draft create --debug` to see a list of the files that are identified as that language.
2. If you see files that you didn't write, consider moving the files into one of the [paths for vendored code][vendor.yml], or use the [manual overrides](#overrides) feature to ignore them.
3. If the files are being misclassified, search for [open issues][linguist-issues] to see if anyone else has already reported the issue. Any information you can add, especially links to public repositories, is helpful.
4. If there are no reported issues of this misclassification, [open an issue][linguist-new-issue] and include a link to the repository or a sample of the code that is being misclassified.

## Overrides

Draft supports a number of different custom overrides strategies for language definitions and vendored paths.

### Using gitattributes

Add a `.gitattributes` file to your project and use standard git-style path matchers for the files you want to override to set `linguist-documentation`, `linguist-language`, `linguist-vendored`, and `linguist-generated`. `.gitattributes` will be used to determine language statistics and (if uploaded to Github) will be used to syntax highlight files.

```
$ cat .gitattributes
*.rb linguist-language=Duck
```

#### Vendored code

Checking code you didn't write, such as JavaScript libraries, into your git repo is a common practice, but this often inflates your project's language stats and may even cause your project to be labeled as another language. By default, Linguist treats all of the paths defined in [lib/linguist/vendor.yml][vendor.yml] as vendored and therefore doesn't include them in the language statistics for a repository.

Use the `linguist-vendored` attribute to vendor or un-vendor paths.

```
$ cat .gitattributes
special-vendored-path/* linguist-vendored
jquery.js linguist-vendored=false
```

#### Documentation

Just like vendored files, Linguist excludes documentation files from your project's language stats. [lib/linguist/documentation.yml][documentation.yml] lists common documentation paths and excludes them from the language statistics for your repository.

Use the `linguist-documentation` attribute to mark or unmark paths as documentation.

```
$ cat .gitattributes
project-docs/* linguist-documentation
docs/formatter.rb linguist-documentation=false
```

#### Generated code

Not all plain text files are true source files. Generated files like minified js and compiled CoffeeScript can be detected and excluded from language stats.

```
$ cat .gitattributes
Api.elm linguist-generated=true
```


[documentation.yml]: https://github.com/github/linguist/blob/master/lib/linguist/documentation.yml
[linguist-issues]: https://github.com/github/linguist/issues
[linguist-new-issue]: https://github.com/github/linguist/issues/new
[vendor.yml]: https://github.com/github/linguist/blob/master/lib/linguist/vendor.yml
