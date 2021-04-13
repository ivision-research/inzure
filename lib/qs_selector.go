package inzure

import "fmt"

type QSSelector struct {
	Resource  string
	Condition *QSCondition
}

func (qss *QSSelector) String() string {
	if qss.Condition == nil {
		return "/" + qss.Resource
	}
	return fmt.Sprintf("/%s[%s]", qss.Resource, qss.Condition.String())
}

func (qss *QSSelector) Contains(o *QSSelector) bool {
	if o == nil {
		return false
	} else if qss.Resource != o.Resource {
		return false
	} else if qss.Condition == nil {
		return true
	} else if o.Condition == nil {
		return false
	} else {
		return qss.Condition.Equals(o.Condition)
	}
}

func (qss *QSSelector) Equals(o *QSSelector) bool {
	if o == nil {
		return false
	}
	if qss.Condition == nil {
		return o.Condition == nil
	}
	return qss.Condition.Equals(o.Condition)
}
