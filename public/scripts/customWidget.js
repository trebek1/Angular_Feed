/**
 * @ngdoc module
 * @name Heelix.CustomWidget
 *
 * @description
 *
 *
 */
angular.module('Heelix.CustomWidget', [




/**
 * @ngdoc controller
 * @name customWidgetController
 *
 * @description
 *
 *
 */
]).controller('customWidgetController', ['$scope', '$http', function($scope, $http) {
    documents = [];

    function pullData(){
    	$http.post("http://localhost:8081/api/all_entity_info").success(function(response){
    		$scope.information = response.LatestNews
    	
    	for(var i =0; i<20; i++){
    		documents.push(response.LatestNews[i].Document)
    		
    	}});
	}
	pullData(); 
	


	
    /*
     *
     * Custom widget business logic should go here.
     *
     */
     // Save the documents array to access in the view 
     $scope.documents = documents; 

    // this is our demo test, just to make sure everything's wired up properly. feel free to
    // delete it once you get going.
    $scope.demoTitle = 'This is a demo headline!';

}]);