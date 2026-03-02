package cache

import (
	"sync"
	"time"

	"github.com/kk-alert/backend/internal/models"
)

// RuleCache provides thread-safe caching for enabled rules.
// Optimized for 1000+ concurrent rule evaluations.
type RuleCache struct {
	mu         sync.RWMutex
	rules      []models.Rule
	lastUpdate time.Time
	ttl        time.Duration
}

// NewRuleCache creates a new rule cache with specified TTL.
func NewRuleCache(ttl time.Duration) *RuleCache {
	return &RuleCache{
		rules: make([]models.Rule, 0),
		ttl:   ttl,
	}
}

// Get returns cached rules if valid, or nil if expired.
func (c *RuleCache) Get() []models.Rule {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if time.Since(c.lastUpdate) > c.ttl {
		return nil
	}
	// Return copy to prevent external modification
	rulesCopy := make([]models.Rule, len(c.rules))
	copy(rulesCopy, c.rules)
	return rulesCopy
}

// Set updates the cached rules.
func (c *RuleCache) Set(rules []models.Rule) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.rules = make([]models.Rule, len(rules))
	copy(c.rules, rules)
	c.lastUpdate = time.Now()
}

// Invalidate clears the cache.
func (c *RuleCache) Invalidate() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.rules = make([]models.Rule, 0)
	c.lastUpdate = time.Time{}
}

// TemplateCache provides thread-safe caching for templates.
type TemplateCache struct {
	mu         sync.RWMutex
	templates  map[uint]models.Template
	lastUpdate time.Time
	ttl        time.Duration
}

// NewTemplateCache creates a new template cache.
func NewTemplateCache(ttl time.Duration) *TemplateCache {
	return &TemplateCache{
		templates: make(map[uint]models.Template),
		ttl:       ttl,
	}
}

// Get returns a cached template by ID.
func (c *TemplateCache) Get(id uint) (models.Template, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if time.Since(c.lastUpdate) > c.ttl {
		return models.Template{}, false
	}

	tpl, ok := c.templates[id]
	if !ok {
		return models.Template{}, false
	}
	return tpl, true
}

// GetDefault returns the default template.
func (c *TemplateCache) GetDefault() (models.Template, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if time.Since(c.lastUpdate) > c.ttl {
		return models.Template{}, false
	}

	for _, tpl := range c.templates {
		if tpl.IsDefault {
			return tpl, true
		}
	}
	return models.Template{}, false
}

// Set updates the cache with templates.
func (c *TemplateCache) Set(templates []models.Template) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.templates = make(map[uint]models.Template, len(templates))
	for _, tpl := range templates {
		c.templates[tpl.ID] = tpl
	}
	c.lastUpdate = time.Now()
}

// ChannelCache provides thread-safe caching for channels.
type ChannelCache struct {
	mu         sync.RWMutex
	channels   map[uint]models.Channel
	lastUpdate time.Time
	ttl        time.Duration
}

// NewChannelCache creates a new channel cache.
func NewChannelCache(ttl time.Duration) *ChannelCache {
	return &ChannelCache{
		channels: make(map[uint]models.Channel),
		ttl:      ttl,
	}
}

// Get returns a cached channel by ID.
func (c *ChannelCache) Get(id uint) (models.Channel, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if time.Since(c.lastUpdate) > c.ttl {
		return models.Channel{}, false
	}

	ch, ok := c.channels[id]
	if !ok {
		return models.Channel{}, false
	}
	return ch, true
}

// Set updates the cache with channels.
func (c *ChannelCache) Set(channels []models.Channel) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.channels = make(map[uint]models.Channel, len(channels))
	for _, ch := range channels {
		c.channels[ch.ID] = ch
	}
	c.lastUpdate = time.Now()
}

// Global cache instances with 30-second TTL for high concurrency scenarios.
var (
	Rules     = NewRuleCache(30 * time.Second)
	Templates = NewTemplateCache(30 * time.Second)
	Channels  = NewChannelCache(30 * time.Second)
)
