package promtop

type Data interface {
	GetCpu(string) []float64
	GetInstances() []string
}

type Cache struct {
	Data
  instances []string
}

func (c *Cache) GetInstances() []string {
  if c.instances == nil {
    c.instances = c.Data.GetInstances()
  }
  return c.instances
}

func (c *Cache) NumberOfInstances() int {
  return len(c.GetInstances())
}

func (c *Cache) MaxInstanceNameLen() int {
  maxInstanceNameLen := 0
  for _, name := range c.GetInstances() {
    if len(name) > maxInstanceNameLen {
      maxInstanceNameLen = len(name)
    }
  }
  return maxInstanceNameLen
}

func (c *Cache) clear() {
  c.instances = nil
}
