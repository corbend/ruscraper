(function(global, angular) {

	angular.module('App').controller('LoginUserCtrl', function($scope, $http, $filter) {
		$scope.login = function(user) {
			$http.post('/login', user).success(function(resp) {
				window.location.href = "/users/" + resp.Id;
			});
		}
	})
})(window, angular);