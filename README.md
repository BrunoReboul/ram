# RAM Real-time Asset Monitor

[Product overview](https://github.com/BrunoReboul/ram-config-template/blob/master/docs/product_overview.md)

## RAM Testing framework

### RAM Unit testing framework

- GO unit testing functions MUST be prefixed with `TestUnit` enabling filtering in [testing_unit.yaml](testing_unit.yaml)
- GO unit test source code file SHOULD be named after the code source file using the suffix `_unit_test.go` example
  - Code source file `func_validatestruct.go`
  - Unit tests code file `func_validatestruct_unit_test.go`
- Unit tests MUST be small enough so all the unit tests are run on each push and each PR update: [testing_unit.yaml](testing_unit.yaml)
- Unit test MUST run in isolation:
  1. No real call to a dependency API
     - meaning, the service account used to run the CI pipeline need no expertanl API IAM related role
  2. Test double (mocks / stubs/ fakes) are not encouraged:
     - prefer to use a real object
     - As calling real object is not allowed in unit test (previous guideline), move these tests to small integration tests (next section)

### RAM Integration testing framework

- GO integration testing functions MUST be prefixed with `TestInteg` enabling filtering in [testing_integ.yaml](testing_integ.yaml)
- GO integration test source code file SHOULD be named after the code source file using the suffix `_integ_test.go` example
  - Code source file `meth_folderdeployment_deploy.go`
  - Unit tests code file `meth_folderdeployment_deploy_integ_test.go`
- Integration tests MUST be small enough so a GO package integration tests are run on each push on the package code and each PR to master update: [testing_integ.yaml](testing_integ.yaml)
- The project hosting the integration tests MUST have a project name that contains `ram-build` avoiding resources deletion, creation driven during integration test to occur in the wrong project
- The project hosting the integration tests MUST have a Cloud Operation Workspace hosted by itself
- The following IAM bindings MUST be set with the service account used to run integration tests:
  - sandboxes folder
    - folder admin
      - required by: grm
  - project hosting integration tests
    - editor
      - required to create, delete integration test resources
- The project hosting the service account used to run the integration test MUST have the following API enabled:
  - Cloud Resource Manager API

## What's next

[RAM microservice packages on go.dev](https://pkg.go.dev/github.com/BrunoReboul/ram)
