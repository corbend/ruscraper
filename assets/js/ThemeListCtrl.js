(function(global, angular) {

	angular.module('App').controller('ThemeListCtrl', function($scope, $http, $filter) {

		$scope.themes = [];
		$scope.blocks = {themes: {loading: false}};
		$scope.ok = true;

		$scope.date = null;
		$scope.time = null;

		numberFormat = function(v) {
			cv = String($filter('number', 2)(v / 100)).split(".")[1];
			return (cv && cv.length == 2) ? cv: (cv + "00").slice(0, 2);
		}

		$scope.printDate = function() {
			var d = new Date();
			$scope.date = d.getFullYear() + "-" + numberFormat(d.getMonth() + 1) + "-" + numberFormat(d.getDate());
			$scope.time = numberFormat(d.getHours()) + ":" + numberFormat(d.getMinutes()) + ":" + numberFormat(d.getSeconds());			
		}

		$scope.printDate();
		setInterval(function() {
			$scope.printDate();
			$scope.$apply();
		}, 1000)

		$scope.runParse = function(url) {
			$scope.ok = false;
			$scope.blocks.themes.loading = true;
			$http.post('/parse', {url: url}).success(function(resp) {
				console.log(resp);
				$scope.themes = resp;
				$scope.ok = true;
				$scope.blocks.themes.loading = false; 
			});
		}

		$scope.chart = null;

		$scope.drawChart = function() {

			var ctx = document.getElementById("parseActionChart");

			if (!$scope.chart) {
				$scope.chart = new Chart(ctx, {
				    type: 'bar',
				    data: {
				        labels: ["Red", "Blue", "Yellow", "Green", "Purple", "Orange"],
				        datasets: [{
				            label: '# of Votes',
				            data: [12, 19, 3, 5, 2, 3]
				        }]
				    },
				    options: {
				        scales: {
				            yAxes: [{
				                ticks: {
				                	beginAtZero:true
				            	}
				        	}]
				    	}
					}
				});
			}
		}

		$scope.getUrls = function() {
			return $http.get('/parse_urls').success(function(resp) {
				$scope.parseUrls = resp.parse_urls;
			})
		}

		$scope.getUrls();

		$scope.getStat = function() {

			return $http.get('/stat').success(function(resp) {

				$scope.stat = resp;
				$scope.drawChart();

			})
		}

		setInterval(function() {
			$scope.getStat().success(function() {
				console.log("stat ok");
			})
		}, 2000)
	})

})(window, angular)