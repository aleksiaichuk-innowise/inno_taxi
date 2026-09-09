package http

import (
	"time"

	"github.com/gofiber/fiber/v2"
)

const dateLayout = "2006-01-02"

// parseDateRange reads ?from=&to= (YYYY-MM-DD), defaulting to the last 30
// days when either is omitted.
func parseDateRange(c *fiber.Ctx) (from, to time.Time, err error) {
	now := time.Now().UTC()
	from = now.AddDate(0, 0, -30)
	to = now

	if raw := c.Query("from"); raw != "" {
		from, err = time.Parse(dateLayout, raw)
		if err != nil {
			return time.Time{}, time.Time{}, err
		}
	}
	if raw := c.Query("to"); raw != "" {
		to, err = time.Parse(dateLayout, raw)
		if err != nil {
			return time.Time{}, time.Time{}, err
		}
		// "to" is inclusive of the whole day from the caller's point of
		// view, but queries compare with `<`, so push it to the start of
		// the next day.
		to = to.AddDate(0, 0, 1)
	}

	return from, to, nil
}
