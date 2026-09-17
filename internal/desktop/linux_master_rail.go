//go:build linux

package desktop

const (
	linuxMasterRailX           = 14
	linuxMasterRailWidth       = 166
	linuxMasterContentLeft     = 204
	linuxMasterRailCardH       = 46
	linuxMasterRailCardGap     = 8
	linuxMasterRailUtilityH    = 38
	linuxMasterRailUtilityGap  = 7
	linuxMasterRailLanguageH   = 30
	linuxMasterRailPrimaryTop  = 64
	linuxMasterRailBottomInset = 14
)

type linuxMasterRailLayout struct {
	brand       linuxRect
	files       linuxRect
	connections linuxRect
	transfers   linuxRect
	settings    linuxRect
	bookmarks   linuxRect
	diagnostics linuxRect
	about       linuxRect
	language    linuxRect
	content     linuxRect
}

func buildLinuxMasterRailLayout(width, height int) linuxMasterRailLayout {
	if width < premiumMinWidth {
		width = premiumMinWidth
	}
	if height < premiumMinHeight {
		height = premiumMinHeight
	}

	layout := linuxMasterRailLayout{
		brand:   linuxRectWH(linuxMasterRailX, 14, linuxMasterRailWidth, 34),
		content: linuxRect{left: linuxMasterContentLeft, top: 0, right: width - premiumOuterGap, bottom: height},
	}

	y := linuxMasterRailPrimaryTop
	layout.files = linuxRectWH(linuxMasterRailX, y, linuxMasterRailWidth, linuxMasterRailCardH)
	y += linuxMasterRailCardH + linuxMasterRailCardGap
	layout.connections = linuxRectWH(linuxMasterRailX, y, linuxMasterRailWidth, linuxMasterRailCardH)
	y += linuxMasterRailCardH + linuxMasterRailCardGap
	layout.transfers = linuxRectWH(linuxMasterRailX, y, linuxMasterRailWidth, linuxMasterRailCardH)
	y += linuxMasterRailCardH + linuxMasterRailCardGap
	layout.settings = linuxRectWH(linuxMasterRailX, y, linuxMasterRailWidth, linuxMasterRailCardH)
	y += linuxMasterRailCardH

	utilityHeight := 3*linuxMasterRailUtilityH + 3*linuxMasterRailUtilityGap + linuxMasterRailLanguageH
	utilityY := height - linuxMasterRailBottomInset - utilityHeight
	if minimum := y + 18; utilityY < minimum {
		utilityY = minimum
	}
	layout.bookmarks = linuxRectWH(linuxMasterRailX, utilityY, linuxMasterRailWidth, linuxMasterRailUtilityH)
	utilityY += linuxMasterRailUtilityH + linuxMasterRailUtilityGap
	layout.diagnostics = linuxRectWH(linuxMasterRailX, utilityY, linuxMasterRailWidth, linuxMasterRailUtilityH)
	utilityY += linuxMasterRailUtilityH + linuxMasterRailUtilityGap
	layout.about = linuxRectWH(linuxMasterRailX, utilityY, linuxMasterRailWidth, linuxMasterRailUtilityH)
	utilityY += linuxMasterRailUtilityH + linuxMasterRailUtilityGap
	layout.language = linuxRectWH(linuxMasterRailX, utilityY, linuxMasterRailWidth, linuxMasterRailLanguageH)
	return layout
}

func linuxMasterTransferBadge(jobs []model.TransferJob) int {
	return relevantTransferCount(jobs)
}
