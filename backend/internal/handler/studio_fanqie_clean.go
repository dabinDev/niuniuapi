package handler

import (
	"context"
	"strings"
)

func fanqieBookHasPrivateUseText(book fanqieBook) bool {
	if containsPrivateUseRune(book.Title) ||
		containsPrivateUseRune(book.Author) ||
		containsPrivateUseRune(book.Category) ||
		containsPrivateUseRune(book.Description) {
		return true
	}
	for _, tag := range book.Tags {
		if containsPrivateUseRune(tag) {
			return true
		}
	}
	return false
}

func mergeFanqieCleanSummary(original, summary fanqieBook) fanqieBook {
	clean := summary
	clean.Rank = original.Rank
	if strings.TrimSpace(original.Score) != "" {
		clean.Score = original.Score
	}
	if strings.TrimSpace(clean.ID) == "" {
		clean.ID = original.ID
	}
	if strings.TrimSpace(clean.Title) == "" {
		clean.Title = original.Title
	}
	if strings.TrimSpace(clean.Author) == "" {
		clean.Author = original.Author
	}
	if strings.TrimSpace(clean.Category) == "" {
		clean.Category = original.Category
	}
	if strings.TrimSpace(clean.Status) == "" {
		clean.Status = original.Status
	}
	if strings.TrimSpace(clean.WordCount) == "" {
		clean.WordCount = original.WordCount
	}
	if strings.TrimSpace(clean.Description) == "" {
		clean.Description = original.Description
	}
	if strings.TrimSpace(clean.CoverURL) == "" {
		clean.CoverURL = original.CoverURL
	}
	if strings.TrimSpace(clean.SourceURL) == "" {
		clean.SourceURL = original.SourceURL
	}
	if len(clean.Tags) == 0 {
		clean.Tags = original.Tags
	}
	return clean
}

func safeFanqieObfuscatedBookFallback(book fanqieBook) fanqieBook {
	clean := book
	if containsPrivateUseRune(clean.Title) {
		id := strings.TrimSpace(clean.ID)
		if id == "" {
			id = strings.TrimSpace(fanqieBookID(clean))
		}
		if id != "" {
			clean.Title = "番茄作品 " + id
		} else {
			clean.Title = "番茄榜单作品"
		}
	}
	if containsPrivateUseRune(clean.Author) || strings.TrimSpace(clean.Author) == "" {
		clean.Author = "番茄小说"
	}
	if containsPrivateUseRune(clean.Category) || strings.TrimSpace(clean.Category) == "" {
		clean.Category = "榜单作品"
	}
	if containsPrivateUseRune(clean.Description) || strings.TrimSpace(clean.Description) == "" {
		clean.Description = "官方详情暂时不可用，已隐藏番茄加密字体字段；可打开作品页或稍后刷新获取完整信息。"
	}
	if len(clean.Tags) > 0 {
		tags := make([]string, 0, len(clean.Tags))
		for _, tag := range clean.Tags {
			if strings.TrimSpace(tag) == "" || containsPrivateUseRune(tag) {
				continue
			}
			tags = append(tags, tag)
		}
		clean.Tags = tags
	}
	return clean
}

func enrichFanqieObfuscatedRankBooks(ctx context.Context, books []fanqieBook) []fanqieBook {
	if len(books) == 0 || fetchFanqieBookSummary == nil {
		return books
	}
	out := make([]fanqieBook, len(books))
	copy(out, books)
	for i, book := range out {
		if !fanqieBookHasPrivateUseText(book) {
			continue
		}
		summary, err := fetchFanqieBookSummary(ctx, book)
		if err != nil || fanqieBookHasPrivateUseText(summary) {
			out[i] = safeFanqieObfuscatedBookFallback(book)
			continue
		}
		out[i] = mergeFanqieCleanSummary(book, summary)
	}
	return out
}
