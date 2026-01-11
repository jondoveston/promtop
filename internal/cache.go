package promtop

type Data interface {
	GetCpu(string) []float64
	GetNodes() []string
	Check() error
	GetType() string // Returns "prometheus" or "node_exporter"
}

type Cache struct {
	Data
	nodes []string
}

func (c *Cache) GetNodes() []string {
	if c.nodes == nil {
		c.nodes = c.Data.GetNodes()
	}
	return c.nodes
}

func (c *Cache) NumberOfNodes() int {
	return len(c.GetNodes())
}

func (c *Cache) MaxNodeNameLen() int {
	maxNodeNameLen := 0
	for _, name := range c.GetNodes() {
		if len(name) > maxNodeNameLen {
			maxNodeNameLen = len(name)
		}
	}
	return maxNodeNameLen
}

func (c *Cache) clear() {
	c.nodes = nil
}
