# Examples

This directory contains two example Rust projects which can be used to test the application.

The projects need to be archived as zip archives. In linux that can be done with the following command (you need to have `zip` installed).

```
cd <your-project-directory>
zip myCodeArchive.zip -r .
```

In `rust-projects` you can find two sample rust projects with all the necessary source files, Cargo files, and the required directory structure.

In `archived-rust-projects` you can find the archives built from the projects in `rust-projects`.

For example to make a zip archive for the `sample` project you can do the following:

```
clone https://github.com/luchev/distributed-testing-framework-for-rust
cd distributed-testing-framework-for-rust/examples/rust-projects/sample
zip sample.zip -r .
```

This will clone the repo, change the working directory to the `sample` project directory and archive it. The resulting `sample.zip` file can be provided to the main app to be tested.
