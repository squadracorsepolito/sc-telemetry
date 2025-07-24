package pkg

import (
	"os"

	"github.com/squadracorsepolito/acmelib"
)

func GetMessagesFromDBC(file string) ([]*acmelib.Message, error) {
	dbcFile, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer dbcFile.Close()
	bus, err := acmelib.ImportDBCFile("bus", dbcFile)
	if err != nil {
		return nil, err
	}

	messages := []*acmelib.Message{}

	for _, nodeInt := range bus.NodeInterfaces() {
		for _, msg := range nodeInt.SentMessages() {
			messages = append(messages, msg)
		}
	}

	return messages, nil
}
