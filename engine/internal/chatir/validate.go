package chatir

import "fmt"

func (c *ChatIR) Normalize() {
	c.Metadata.MessageCount = len(c.Messages)
}

func (c *ChatIR) Validate() error {
	if c.Platform == "" {
		return fmt.Errorf("platform is required")
	}
	if c.ConversationID == "" {
		return fmt.Errorf("conversation_id is required")
	}
	if len(c.Participants) == 0 {
		return fmt.Errorf("participants are required")
	}

	c.Normalize()
	return nil
}
