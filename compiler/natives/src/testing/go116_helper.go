// +build js
// +build go1.16

package testing

// Helper may be called simultaneously from multiple goroutines.
func (c *common) Helper() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.helperPCs == nil {
		c.helperPCs = make(map[uintptr]struct{})
	}
	c.helperPCs[0] = struct{}{}
	c.helperNames = nil // map will be recreated next time it is needed
}
