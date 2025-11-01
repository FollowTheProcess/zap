#!/usr/bin/env bash

# This is an integration test suite for the entire program, written in [bats](https://bats-core.readthedocs.io/en/stable/index.html)

setup() {
    export BATS_LIB_PATH=${BATS_LIB_PATH:-"/usr/lib"}
    bats_load_library bats-support
    bats_load_library bats-assert
}

@test "help works" {
    run ./bin/zap --help
    assert_success
    assert_output --partial "A command line .http file toolkit"
}

@test "version works" {
    run ./bin/zap --version
    assert_success
    # test is set as an ldflag from Taskfile.yml
    assert_output --partial "Version: test"
}
