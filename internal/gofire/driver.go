package gofire

type StorageDriver int

const (
	Postgres StorageDriver = iota + 1
	Redis
)
