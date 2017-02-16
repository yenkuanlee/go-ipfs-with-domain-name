#!/bin/sh

test_description="Tests for various fixed issues and regressions."

. lib/test-lib.sh

test_init_ipfs

test_launch_ipfs_daemon

# Tests go here

test_expect_success "commands command with flag flags works via HTTP API - #2301" '
	curl "http://$API_ADDR/api/v0/commands?flags" | grep "verbose"
'

test_expect_success "ipfs refs local over HTTP API returns NDJOSN not flat - #2803" '
	echo "Hello World" | ipfs add &&
	curl "http://$API_ADDR/api/v0/refs/local" | grep "Ref" | grep "Err"
'

test_expect_success "args expecting stdin dont crash when not given" '
	curl "$API_ADDR/api/v0/bootstrap/add" > result
'

test_expect_success "no panic traces on daemon" '
	test_must_fail grep "nil pointer dereference" daemon_err
'

test_kill_ipfs_daemon

test_expect_success "ipfs daemon --offline --mount fails - #2995" '
	test_expect_code 1 ipfs daemon --offline --mount 2>daemon_err &&
	grep "mount is not currently supported in offline mode" daemon_err
'

test_done

