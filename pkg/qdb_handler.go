package pkg

import (
	"context"

	"github.com/squadracorsepolito/acmetel/can"
	"github.com/squadracorsepolito/acmetel/questdb"
)

type QuestDBHandler struct{}

func NewQuestDBHandler() *QuestDBHandler {
	return &QuestDBHandler{}
}

func (h *QuestDBHandler) Init(_ context.Context) error {
	return nil
}

func (h *QuestDBHandler) Handle(_ context.Context, canMsgBatch *can.Message) (*questdb.Result, error) {
	rows := make([]*questdb.Row, 0, canMsgBatch.SignalCount)

	for _, sig := range canMsgBatch.Signals {
		table := sig.Table

		row := questdb.NewRow(table.String())

		row.AddColumn(questdb.NewSymbolColumn("name", sig.Name))

		// For enum signals, since it uses a symbol column, we need to add the can_id and raw_value
		// columns after the enum_value column
		if table != can.CANSignalTableEnum {
			row.AddColumn(questdb.NewIntColumn("can_id", sig.CANID))
			row.AddColumn(questdb.NewIntColumn("raw_value", sig.RawValue))
		}

		switch table {
		case can.CANSignalTableFlag:
			row.AddColumn(questdb.NewBoolColumn("flag_value", sig.ValueFlag))

		case can.CANSignalTableInt:
			row.AddColumn(questdb.NewIntColumn("integer_value", sig.ValueInt))

		case can.CANSignalTableFloat:
			row.AddColumn(questdb.NewFloatColumn("float_value", sig.ValueFloat))

		case can.CANSignalTableEnum:
			row.AddColumn(questdb.NewSymbolColumn("enum_value", sig.ValueEnum))
			row.AddColumn(questdb.NewIntColumn("can_id", sig.CANID))
			row.AddColumn(questdb.NewIntColumn("raw_value", sig.RawValue))
		}

		rows = append(rows, row)
	}

	return questdb.NewResult(rows...), nil
}

func (h *QuestDBHandler) Close() {}
