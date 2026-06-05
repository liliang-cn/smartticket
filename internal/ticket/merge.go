package ticket

import (
	"context"
	stderrors "errors"
	"fmt"

	"github.com/company/smartticket/internal/authz"
	"github.com/company/smartticket/internal/errors"
	"github.com/company/smartticket/internal/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// validLinkTypes is the exhaustive set of allowed TicketLink.Type values.
var validLinkTypes = map[string]bool{
	"related":   true,
	"duplicate": true,
	"blocks":    true,
}

// LinkResponse is the API representation of a TicketLink, enriched with the
// other ticket's summary (id/title/status) for display purposes.
type LinkResponse struct {
	ID          uint   `json:"id"`
	SourceID    uint   `json:"source_id"`
	TargetID    uint   `json:"target_id"`
	Type        string `json:"type"`
	OtherTicket struct {
		ID     uint   `json:"id"`
		Title  string `json:"title"`
		Status string `json:"status"`
	} `json:"other_ticket"`
}

// Merge consolidates sourceID into targetID (team-only, transactional).
//
// Rules enforced:
//   - team-only (customer actors are rejected)
//   - source != target (no self-merge)
//   - source must not already be merged
//   - target must not already be merged (don't chain merges)
//
// Transaction steps:
//  1. Reassign all Message rows from source to target
//  2. Reassign all Attachment rows from source to target
//  3. Set source.Status = "merged", source.MergedIntoID = targetID
//  4. Record a TicketEvent on source ("merged into #target")
//  5. Record a TicketEvent on target ("absorbed #source")
func (s *Service) Merge(actor authz.Actor, sourceID, targetID uint) error {
	if !actor.IsTeam() {
		return errors.NewForbiddenError("only team members can merge tickets")
	}
	if sourceID == targetID {
		return errors.NewBusinessRuleError("merge_self", "cannot self-merge a ticket")
	}

	src, err := s.findTicketForActor(actor, sourceID)
	if err != nil {
		return err
	}
	tgt, err := s.findTicketForActor(actor, targetID)
	if err != nil {
		return err
	}

	if src.Status == "merged" {
		return errors.NewBusinessRuleError("merge_already_merged", "source ticket is already merged")
	}
	if tgt.Status == "merged" {
		return errors.NewBusinessRuleError("merge_into_merged", "cannot merge into a ticket that is already merged")
	}

	txErr := s.db.Transaction(func(tx *gorm.DB) error {
		// Reassign messages from source to target.
		if err := tx.Model(&models.Message{}).
			Where("ticket_id = ?", sourceID).
			Update("ticket_id", targetID).Error; err != nil {
			return fmt.Errorf("reassign messages: %w", err)
		}

		// Reassign attachments from source to target.
		if err := tx.Model(&models.Attachment{}).
			Where("ticket_id = ?", sourceID).
			Update("ticket_id", targetID).Error; err != nil {
			return fmt.Errorf("reassign attachments: %w", err)
		}

		// Mark source as merged.
		if err := tx.Model(&models.Ticket{}).Where("id = ?", sourceID).
			Updates(map[string]interface{}{
				"status":         "merged",
				"merged_into_id": targetID,
			}).Error; err != nil {
			return fmt.Errorf("update source ticket: %w", err)
		}

		// Record events on both tickets (inside the transaction so a failure
		// rolls back the whole merge rather than silently losing audit trail).
		if err := tx.Create(&models.TicketEvent{
			TicketID: sourceID,
			UserID:   actor.UserID,
			Action:   "merged",
			Summary:  fmt.Sprintf("merged into #%d: %s", tgt.ID, tgt.Title),
		}).Error; err != nil {
			return fmt.Errorf("record source event: %w", err)
		}
		if err := tx.Create(&models.TicketEvent{
			TicketID: targetID,
			UserID:   actor.UserID,
			Action:   "merged",
			Summary:  fmt.Sprintf("absorbed #%d: %s", src.ID, src.Title),
		}).Error; err != nil {
			return fmt.Errorf("record target event: %w", err)
		}

		return nil
	})
	if txErr != nil {
		return fmt.Errorf("merge failed: %w", txErr)
	}

	// Best-effort notification (never affects the merge outcome).
	if s.notifier != nil && tgt.CustomerID != nil {
		s.notifier.Notify(
			context.Background(),
			s.customerRecipients(*tgt.CustomerID, actor.UserID),
			"ticket_status",
			fmt.Sprintf("Ticket #%d was merged into #%d", src.ID, tgt.ID),
			"",
			"ticket",
			tgt.ID,
		)
	}

	return nil
}

// LinkTickets creates a TicketLink from sourceID to targetID with the given
// linkType. If the exact (source, target, type) triple already exists, the
// existing record is returned (idempotent).
func (s *Service) LinkTickets(actor authz.Actor, sourceID, targetID uint, linkType string) (*models.TicketLink, error) {
	if !actor.IsTeam() {
		return nil, errors.NewForbiddenError("only team members can link tickets")
	}
	if sourceID == targetID {
		return nil, errors.NewBusinessRuleError("link_self", "cannot self-link a ticket")
	}
	if !validLinkTypes[linkType] {
		return nil, errors.NewValidationError("invalid link type: must be one of related, duplicate, blocks")
	}

	// Verify both tickets are accessible to the actor (prevents existence leaks).
	if _, err := s.findTicketForActor(actor, sourceID); err != nil {
		return nil, err
	}
	if _, err := s.findTicketForActor(actor, targetID); err != nil {
		return nil, err
	}

	link := &models.TicketLink{
		SourceID: sourceID,
		TargetID: targetID,
		Type:     linkType,
	}

	// ON CONFLICT DO NOTHING: if the unique (source,target,type) triple already
	// exists the insert is silently skipped and RowsAffected == 0.
	result := s.db.Clauses(clause.OnConflict{DoNothing: true}).Create(link)
	if result.Error != nil {
		return nil, fmt.Errorf("create ticket link: %w", result.Error)
	}

	// Row was not inserted (duplicate) — fetch and return the existing record.
	if result.RowsAffected == 0 {
		if err := s.db.Where("source_id = ? AND target_id = ? AND type = ?",
			sourceID, targetID, linkType).First(link).Error; err != nil {
			return nil, fmt.Errorf("find existing link: %w", err)
		}
	}

	return link, nil
}

// Unlink deletes a TicketLink by ID (team-only).
//
// ticketID is the ticket the caller is acting on; the link must have ticketID
// as either its source or target — otherwise NotFound is returned. This
// prevents a team member from deleting an arbitrary link by guessing its ID.
func (s *Service) Unlink(actor authz.Actor, ticketID, linkID uint) error {
	if !actor.IsTeam() {
		return errors.NewForbiddenError("only team members can unlink tickets")
	}

	// Ensure the ticket is visible to this actor (customer isolation + soft-delete).
	if _, err := s.findTicketForActor(actor, ticketID); err != nil {
		return err
	}

	// Load the link and verify it belongs to the given ticket.
	var link models.TicketLink
	if err := s.db.First(&link, linkID).Error; err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return errors.NewNotFoundError("ticket link")
		}
		return fmt.Errorf("load ticket link: %w", err)
	}
	if link.SourceID != ticketID && link.TargetID != ticketID {
		// The link exists but is not associated with the given ticket — treat
		// as not found so callers cannot enumerate links by ID.
		return errors.NewNotFoundError("ticket link")
	}

	result := s.db.Delete(&models.TicketLink{}, linkID)
	if result.Error != nil {
		return fmt.Errorf("delete ticket link: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return errors.NewNotFoundError("ticket link")
	}
	return nil
}

// ListLinks returns all TicketLinks where the given ticketID is either the
// source or the target. Each link is enriched with the other ticket's
// id, title, and status for display in the UI.
func (s *Service) ListLinks(actor authz.Actor, ticketID uint) ([]LinkResponse, error) {
	// Verify the actor may see this ticket.
	if _, err := s.findTicketForActor(actor, ticketID); err != nil {
		return nil, err
	}

	var links []models.TicketLink
	if err := s.db.Where("source_id = ? OR target_id = ?", ticketID, ticketID).
		Find(&links).Error; err != nil {
		return nil, fmt.Errorf("list ticket links: %w", err)
	}

	out := make([]LinkResponse, 0, len(links))
	for _, l := range links {
		otherID := l.TargetID
		if l.TargetID == ticketID {
			otherID = l.SourceID
		}

		var other models.Ticket
		if err := s.db.Select("id, title, status").First(&other, otherID).Error; err != nil {
			if stderrors.Is(err, gorm.ErrRecordNotFound) {
				continue // stale link — skip gracefully
			}
			return nil, fmt.Errorf("load other ticket %d: %w", otherID, err)
		}

		lr := LinkResponse{
			ID:       l.ID,
			SourceID: l.SourceID,
			TargetID: l.TargetID,
			Type:     l.Type,
		}
		lr.OtherTicket.ID = other.ID
		lr.OtherTicket.Title = other.Title
		lr.OtherTicket.Status = other.Status
		out = append(out, lr)
	}
	return out, nil
}
