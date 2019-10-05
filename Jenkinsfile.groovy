pipeline {
  agent {
    label "jenkins-maven"
  }
  environment {
    CHART_REPOSITORY= 'https://artifactory.cluster.foxsports-gitops-prod.com.au/artifactory/helm' // Please do not edit this line! Managed by customize.sh
    ORG = 'fsa-streamotion' 
    APP_NAME = 'fsa-streamotion-jx'
  }

//--------

  stages {
    stage('CI Build + PREVIEW') {
      when {
        branch 'feature/local-docker-build'
      }
      environment {
//        PREVIEW_VERSION = get_previewVersion(APP_NAME, BRANCH_NAME, BUILD_NUMBER)
        PREVIEW_NAMESPACE = "NS"
        PREVIEW_VERSION = "0.0.0-SNAPSHOT-$BUILD_NUMBER"

        HELM_RELEASE = "$APP_NAME-$BRANCH_NAME".toLowerCase()
      }
      steps {
//        PREVIEW_VERSION = previewNames("1", BRANCH_NAME, BUILD_NUMBER)["previewName"]
        container('maven') {
          sh "echo **************** PREVIEW_VERSION: $PREVIEW_VERSION , PREVIEW_NAMESPACE: $PREVIEW_NAMESPACE, HELM_RELEASE: $HELM_RELEASE"
//          previewNames("3",BRANCH_NAME, BUILD_NUMBER)

          sh "echo $PREVIEW_VERSION > PREVIEW_VERSION"
          sh "export VERSION=$PREVIEW_VERSION && skaffold build -f skaffold.yaml"
          script {
            def buildVersion =  readFile "${env.WORKSPACE}/PREVIEW_VERSION"
            currentBuild.description = "$APP_NAME.$PREVIEW_NAMESPACE"
          }
        }
      }
    }

    stage('Build Release') {
      when {
        branch 'master'
      }

      steps {
        container('maven') {
              // ensure we're not on a detached head
              sh "git config --global credential.helper store"
              sh "jx step git credentials"

              sh "echo \$(jx-release-version) > VERSION"
              sh "jx step tag --version \$(cat VERSION)"

              sh "export VERSION=`cat VERSION` && skaffold build -f skaffold.yaml"

              script {
                def buildVersion =  readFile "${env.WORKSPACE}/VERSION"
                currentBuild.description = "$buildVersion"
                currentBuild.displayName = "$buildVersion"
              }          
        }
      }
    }
  
    stage('Publish DockerHub Release') {
        agent {
          label "dockerhub-maven"
        }
        when {
          branch 'master'
        }

        steps {
          container('maven') {
                sh "export VERSION=`cat VERSION` && skaffold build -f skaffold-dockerhub.yaml"

                script {
                  def buildVersion =  readFile "${env.WORKSPACE}/VERSION"
                  currentBuild.description = "$buildVersion"
                  currentBuild.displayName = "$buildVersion"
                }          
          }
        }
    }
  }


//--------


  post {
        always {
          cleanWs()
        }
  }
}
