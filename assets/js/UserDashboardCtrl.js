(function(global, angular) {

	angular.module('App').controller('UserDashboardCtrl', function($scope, $http, $filter, $location, websocket) {

		$scope.topics = [];
		$scope.categories = [];
		$scope.activeFilters = [];

		$scope.loadCategories = function() {
			var r = $http.get('/categories') 
			r.success(function(resp) {
				$scope.categories = resp.rows;
			});
			return r;
		}

		$scope.loadTopics = function() {
			var r = $http.get('/topics');
			r.success(function(resp) {
				$scope.topics = resp;
			})
		}

		$scope.loadTopics();

		websocket.setMessageHandler(function(action, payload) {
			if (action == "get_updates") {
				console.log("get updates", payload)
				$scope.subscriptions = payload.Items;
				$scope.applyFilters();
			}
		})

		$scope.applySubscriptions = function(category) {

			var subscribed = [];
			subscribed = Object.keys($scope.categories).map(function(k) {return $scope.categories[k]}).filter(function(item) {
				return item.subscribed;
			});

			if (subscribed.length > 0) {
				var userId = $location.$$absUrl.split("/").slice(-1)[0];
				$http.post('/users/' + userId + '/subscribe', 
					{categories: subscribed.map(function(i) {return i.id})})
				.success(function() {

				});
			} else {
				console.log("select one or more categories to subscribe");
			}
		}

		$scope.addToFavorites = function(subscription) {
			var userId = $location.$$absUrl.split("/").slice(-1)[0];

			$http.post('/subscriptions/' + userId + '/favorites/' + subscription.Id + "?indexName=" + subscription.IndexName, subscription).success(function(resp) {
				console.log("add to favorites");
			});
		}

		$scope.loadFavorites = function() {

			var userId = $location.$$absUrl.split("/").slice(-1)[0];

			var catIds = $scope.subscriptions.map(function(s) {
				return s.CategoryId;
			});

			var indexNames = [];

			$scope.categories.forEach(function(cat) {
				if (catIds.indexOf(cat.Id) != -1) {
					if (cat.IndexName) {
						indexNames.push(cat.indexName);
					}
				}
			})

			var url = '/subscriptions/' + userId + "/favorites";
			if (indexNames.length > 0) {
				url += "?=indexName" + indexNames.join(",");
			}
			
			$http.get(url).success(function(resp) {
				$scope.favorites = resp;
			});
		}

		$scope.loadSubscriptions = function() {
			
			var userId = $location.$$absUrl.split("/").slice(-1)[0];
			var r = $http.get('/subscriptions/' + userId);
			r.success(function(resp) {
				$scope.subscriptions = resp.rows;
				$scope.subscriptions.forEach(function(subs) {
					$scope.categories.forEach(function(cat) {
						if (cat.id == subs.category_id) {
							cat.subscribed = true;
						}
					})
				})				
			});

			return r;
		}

		$scope.loadCounters = function() {
			$http.get('/subscriptions/counters').success(function(resp) {
				resp.forEach(function(category) {
					var matchListCategory = null;
					$scope.categories.forEach(function(cat) {
						if (cat.id == category.id) {
							matchListCategory = cat;
						}
					});

					if (matchListCategory) {
						matchListCategory.counter = category.counter;
					}
				});
			})
		}

		$scope.loadCategories().then(function() {
			$scope.loadSubscriptions().then(function() {
				$scope.loadFavorites();
			});
		});

		$scope.applyFilters = function() {
			if ($scope.activeFilters.length > 0) {
				$scope.filteredSubscriptions = $scope.subscriptions.filter(function(s) {
					var include = false;
					$scope.activeFilters.forEach(function(c) {
						include = include || s.SearchTerms.indexOf(c.name) != -1;
					});
					return include;
				});
			} else {
				$scope.filteredSubscriptions = $scope.subscriptions;
			}
		}

		$scope.filterByCategory = function(cat) {
			if ($scope.activeFilters.indexOf(cat) == -1) {
				$scope.activeFilters.push(cat);				
			} else {
				$scope.activeFilters.splice($scope.activeFilters.indexOf(cat), 1);
			}

			cat.filtered = !cat.filtered;

			$scope.applyFilters();
		}

		setInterval(function() {
			$scope.loadCounters();
		}, 2000);

		setInterval(function() {

			var userId = $location.$$absUrl.split("/").slice(-1)[0];

			websocket.socket.send(JSON.stringify({
				action: 'get_updates',
				payload: {
					user_id: userId
				}
			}))
		}, 2000);
	})

})(window, angular);