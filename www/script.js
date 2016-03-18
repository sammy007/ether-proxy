HandlebarsIntl.registerWith(Handlebars);

$(function() {
	window.state = {};
	var source = $("#stats-template").html();
	var template = Handlebars.compile(source);
	refreshStats(template);

	setInterval(function() {
		refreshStats(template);
	}, 5000)
});

function refreshStats(template) {
	$.getJSON("/stats", function(stats) {
		$("#alert").addClass('hide');

		// Sort miners by ID
		if (stats.miners) {
			stats.miners = stats.miners.sort(compare)
		}

		var epochOffset = (30000 - (stats.height % 30000)) * 1000 * 14.4
		stats.nextEpoch = stats.now + epochOffset

		// Repaint stats
		var html = template(stats);
		$('#stats').html(html);
	}).fail(function() {
		$("#alert").removeClass('hide');
	});
}

function compare(a, b) {
	if (a.name < b.name)
		return -1;
	if (a.name > b.name)
		return 1;
	return 0;
}
