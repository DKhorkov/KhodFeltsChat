package interfaces

//go:generate mockgen -source=controllers.go -destination=../../mocks/controllers/controller.go -package=mockcontrollers -exclude_interfaces=
type Controller interface {
	Run()
	Stop()
}
