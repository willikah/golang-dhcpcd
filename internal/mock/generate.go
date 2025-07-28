package mock

//go:generate go run go.uber.org/mock/mockgen -source=../port/network.go -destination=network_configuration_manager_mock.go -package=mock
//go:generate go run go.uber.org/mock/mockgen -source=../port/infrastructure.go -destination=infrastructure_mocks.go -package=mock
