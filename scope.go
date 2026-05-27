package hal

type scopeImpl struct {
	values map[string]Value
	parent Scope
}

func NewScope(parent Scope) Scope {
	return &scopeImpl{
		values: make(map[string]Value),
		parent: parent,
	}
}

func (s *scopeImpl) Get(name string) Value {
	if val, ok := s.values[name]; ok {
		return val
	}
	if s.parent != nil {
		return s.parent.Get(name)
	}
	return Value{Type: TypeVoid}
}

func (s *scopeImpl) Set(name string, val Value) {
	s.values[name] = val
}

func (s *scopeImpl) Exists(name string) bool {
	if _, ok := s.values[name]; ok {
		return true
	}
	if s.parent != nil {
		return s.parent.Exists(name)
	}
	return false
}
