pipeline {
  agent {
    label "jenkins-maven"
  }
  environment {
    CHART_REPOSITORY= 'https://artifactory.cluster.foxsports-gitops-prod.com.au/artifactory/helm' // Please do not edit this line! Managed by customize.sh
    ORG = 'fsa-streamotion' 
    APP_NAME = 'fsa-streamotion-jx'
  }




  
  post {
        always {
          cleanWs()
        }
  }
}
