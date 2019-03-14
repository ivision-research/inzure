package inzure

import "fmt"

// QSOpT is an operation for a query string
type QSOpT uint8

const (
	QSOpUk QSOpT = iota
	QSOpEq
	QSOpNe
	QSOpGt
	QSOpGte
	QSOpLt
	QSOpLte
	QSOpLike
	QSOpNotLike
)

func (op QSOpT) String() string {
	switch op {
	case QSOpUk:
		return "Unknown"
	case QSOpEq:
		return "=="
	case QSOpNe:
		return "!="
	case QSOpGt:
		return ">"
	case QSOpGte:
		return ">="
	case QSOpLt:
		return "<"
	case QSOpLte:
		return "<="
	case QSOpLike:
		return "LIKE"
	case QSOpNotLike:
		return "!LIKE"
	default:
		return fmt.Sprintf("Unknown Val(%d)", op)
	}
}

type QSArraySelT int

const (
	QSArraySelUk  QSArraySelT = -4
	QSArraySelAny QSArraySelT = -3
	QSArraySelAll QSArraySelT = -2
	QSArraySelLen QSArraySelT = -1
)

func (qsa QSArraySelT) String() string {
	switch qsa {
	case QSArraySelUk:
		return "?"
	case QSArraySelAll:
		return "ALL"
	case QSArraySelAny:
		return "ANY"
	case QSArraySelLen:
		return "LEN"
	default:
		if qsa < 0 {
			return fmt.Sprintf("Bad Selector (%d)", qsa)
		} else {
			return fmt.Sprintf("%d", qsa)
		}
	}
}
