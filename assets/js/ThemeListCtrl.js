(function(global, angular) {

	angular.module('App').controller('ThemeListCtrl', function($scope, $http, $filter, websocket) {

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

		//polling by websocket parsed url statuses
		websocket.setMessageHandler(function(action, payload) {
			if (action == "parse_active") {
				var url;
				$scope.parseUrls.forEach(function(url) {
					if (url == payload.Url) {
						url = payload;
					}
				})

				if (url) {
					var idx = $scope.parseUrls.indexOf(url);
					$scope.parseUrls[idx].active = true;
				}

			} else if (action == "parse_nonactive") {
				var url;
				$scope.parseUrls.forEach(function(url) {
					if (url == payload.Url) {
						url = payload;
					}
				})

				if (url) {
					var idx = $scope.parseUrls.indexOf(url);
					$scope.parseUrls[idx].active = false;
				}
			} else if (action == "new_update") {

			}

			$scope.$apply();
		})

		$scope.chart = null;

		$scope.drawChart = function() {

			if (!$scope.chart) {

				$scope.chart = new Chart(document.getElementById("parseActionChart"), {
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

			if (!$scope.hourNewHits) {

			}

			if (!$scope.elasticStatChart) {

			}

			if (!$scope.parseChart) {

			}

			if (!$scope.redisStatChart) {

			}
		}

		$scope.getUrls = function() {
			return $http.get('/parse_urls').success(function(resp) {
				parseUrls = [];
				resp.parse_urls.forEach(function(u) {
					parseUrls.push({
						url: u,
						active: false
					});
				});

				$scope.parseUrls = parseUrls;
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

		$scope.getCategories = function() {
			return $http.get('/categories').success(function(resp) {
				$scope.categories = resp.categories;
			})
		}

		$scope.getCategories();
		//categories
		$scope.newCategory = {};
		$scope.createCategory = false;

		$scope.toggleCategorySave = function() {
			if (!$scope.createCategory) $scope.createNewCategory();
			else $scope.confirmNewCategorySave();
		}

		$scope.createNewCategory = function() {
			$scope.newCategory = {};
			$scope.createCategory = true;
		}

		$scope.confirmNewCategorySave = function() {
			return $http.post('/categories', $scope.newCategory).success(function(resp) {
				console.log(resp);
				$scope.cancelNewCategory();
				$scope.getCategories();
			})
		}

		$scope.cancelNewCategory = function() {
			$scope.createCategory = false;
		}

		$scope.removeCategory = function(category) {
			return $http({method: 'DELETE', url: '/categories/' + category.id}).success(function(resp) {
				console.log("delete success");

				var removedFilter;

				Object.keys($scope.categories).forEach(function(k, index) {
					var f = $scope.categories[k]
					if (f.id == category.id) {
						removedFilter = k;
					}
				})

				if (removedFilter) {
					delete $scope.categories[removedFilter];
				}
			})
		}

		$scope.elasticStatByIndex = {}

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

				var indexNames = [];
				$scope.queryIndexes.forEach(function(idx) {
					indexNames.push(idx.name);						
		 		});
				$scope.stat.elasticIndexesStats.forEach(function(ind) {					
			 		$scope.elasticStatByIndex[ind.Name] = ind.TotalDocs;
			 		console.log($scope.elasticStatByIndex[ind.Name]);
			 	})

				$scope.drawChart();
			})
		}

		$scope.queryFilters = [
			{name: 'LastDay'}, 
			{name: 'Last5Days'}, 
			{name: 'Last10Days'}, 
			{name: 'LastMonth'}, 
			{name: 'Last6Month'}
		];

		$scope.queryIndexes = [{name: 'programming_books'}, {name: 'programming_videos'}];

		$scope.filters = {
			current: $scope.queryFilters[0],
			queryIndex: $scope.queryIndexes[0]
		}

		$scope.getFavorites = function() {
			return $http.get('/filters/' + $scope.filters.current.name + "?indexName=" + $scope.filters.queryIndex.name).success(function(resp) {
				$scope.favoritesThemes = {};				
				Object.keys(resp).forEach(function(k) {					
					$scope.favoritesThemes[k] = resp[k];
					$scope.favoritesThemes[k].CreateDateStr = (new Date(resp[k].CreateDate * 1000)).toString();
				})
				console.log($scope.favoritesThemes);
			})
		}

		setInterval(function() {
			$scope.getStat().success(function() {
				console.log("stat ok");
			})
		}, 2000)

		setInterval(function() {
			$scope.getFavorites().success(function() {
				console.log("refresh last day themes");
			})
		}, 2000)
	})

})(window, angular)