(function(global, angular) {

	angular.module('App').controller('RegisterUserCtrl', function($scope, $http, $filter) {
		$scope.register = function(user) {
			$http.post('/users', user).success(function() {
				console.log("register new user");
			});
		}
	})
})(window, angular);