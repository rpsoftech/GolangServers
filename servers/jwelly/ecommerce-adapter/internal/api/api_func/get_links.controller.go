package ecommerce_api_func

import (
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/rpsoftech/golang-servers/interfaces"
	"golang.org/x/net/html"
)

type LinkReqBody struct {
	HtmlString string `json:"html"`
}

func GetLinksFromString(c fiber.Ctx) error {
	body := new(LinkReqBody)
	c.Bind().Body(body)
	if body.HtmlString == "" {
		return &interfaces.RequestError{
			StatusCode: fiber.StatusBadRequest,
			Code:       interfaces.ERROR_INVALID_INPUT,
			Message:    "Html String Not Found",
			Name:       "ERROR_INVALID_INPUT",
		}
	}

	imageURLs, err := extractImageURLs(body.HtmlString)
	if err != nil {
		return &interfaces.RequestError{
			StatusCode: fiber.StatusInternalServerError,
			Code:       interfaces.ERROR_INTERNAL_SERVER,
			Message:    "Failed to extract image URLs",
			Name:       "ERROR_INTERNAL_SERVER",
		}
	}
	return c.JSON(fiber.Map{
		"success": true,
		"data":    imageURLs,
	})
}

func extractImageURLs(htmlContent string) ([]string, error) {
	doc, err := html.Parse(strings.NewReader(htmlContent))
	if err != nil {
		return nil, err
	}

	var imageURLs []string
	var traverse func(*html.Node)

	traverse = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "img" {
			for _, attr := range n.Attr {
				if attr.Key == "src" {
					imageURLs = append(imageURLs, attr.Val)
					break
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			traverse(c)
		}
	}

	traverse(doc)
	return imageURLs, nil
}
