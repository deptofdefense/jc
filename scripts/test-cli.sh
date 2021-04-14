#!/bin/bash

# =================================================================
#
# Work of the U.S. Department of Defense, Defense Digital Service.
# Released as open source under the MIT License.  See LICENSE file.
#
# =================================================================

set -euo pipefail

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

export testdata_local="${DIR}/../testdata"

export temp="${DIR}/../temp"

testBlankArray() {
  local input="[]"
  local expected='[]'
  local output=$(echo "${input}" | "${DIR}/../bin/jc" 2>&1)
  assertEquals "unexpected output" "${expected}" "${output}"
}

testBlankObject() {
  local input="{}"
  local expected='{}'
  local output=$(echo "${input}" | "${DIR}/../bin/jc" 2>&1)
  assertEquals "unexpected output" "${expected}" "${output}"
}

testSimple() {
  local input='[{"a":  "b", "c": 1  }]'
  local expected='[{"a":"b","c":1}]'
  local output=$(echo "${input}" | "${DIR}/../bin/jc" 2>&1)
  assertEquals "unexpected output" "${expected}" "${output}"
}

oneTimeSetUp() {
  echo "Using temporary directory at ${SHUNIT_TMPDIR}"
  echo "Reading testdata from ${testdata_local}"
}

oneTimeTearDown() {
  echo "Tearing Down"
}

# Load shUnit2.
. "${DIR}/shunit2"
