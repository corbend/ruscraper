(function(global, angular) {

	angular.module('App').controller('ThemeListCtrl', function($scope, $http, $filter) {

		$scope.themes = [];
		$scope.resultFilters = [];
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

		$scope.getResultFilters = function() {
			return $http.get('/filters').success(function(resp) {
				$scope.resultFilters = resp.filters;
			})
		}

		$scope.getResultFilters();

		$scope.newFilter = {};
		$scope.createFilter = false;

		$scope.toggleFilterSave = function() {
			if (!$scope.createFilter) $scope.createNewFilter();
			else $scope.confirmNewFilterSave();
		}

		$scope.createNewFilter = function() {
			$scope.newFilter = {};
			$scope.createFilter = true;
		}

		$scope.confirmNewFilterSave = function() {
			return $http.post('/filters', $scope.newFilter).success(function(resp) {
				console.log(resp);
			})
		}

		$scope.cancelNewFilter = function() {
			$scope.createFilter = false;
		}

		$scope.applyFilter = function(filter_) {
			return $http.post('/filters/apply', filter_).success(function(resp) {
				$scope.themes = resp;
			})
		}

		$scope.removeFilter = function(filter_) {
			return $http({method: 'DELETE', url: '/filters/' + filter_.id}).success(function(resp) {
				console.log("delete success");

				var removedFilter;

				Object.keys($scope.resultFilters).forEach(function(k, index) {
					var f = $scope.resultFilters[k]
					if (f.id == filter_.id) {
						removedFilter = k;
					}
				})

				if (removedFilter) {
					delete $scope.resultFilters[removedFilter];
				}
			})
		}

		$scope.getStat = function() {

			return $http.get('/stat').success(function(resp) {

				$scope.stat = resp;
				var date = new Date();
				var indexes = ['programming_books', 'programming_videos'];
				var dateStr = String(date.getFullYear()) + "-" + String((date.getMonth() + 1)) + "-" + String(date.getDate()) + "T" + String(date.getHours()) + ":00:00";
				console.log("stat date", dateStr)
				sum = 0;
				indexes.forEach(function(index) {
					sum += Number(resp['new_hits_cnt_' + dateStr + "_" + index]);
				})
				$scope.newCnt = sum;
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