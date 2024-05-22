# testdata/proj1

Test project.

Open a databrowser:

```sh
> databrowser.NewBrowserWindow("testdata/proj1")
```

Procedural flow:

1. `Jobs` shows all the directories under `data` that contain results of simulation runs: one row per directory / Job.  Typically have multiple data files per Job.

2. `Results` grabs specific `.tsv` data files from subset of selected `Jobs` runs: one row per data file.

3. `Plot` plots selected data files in Results.  Allows you to compare various subsets of data files that have been selected into Results.

