pipeline {
    agent {
        label "jenkins-go"
    }
    stages {
        stage('CI Build and Test') {
            when {
                branch 'PR-*'
            }
            steps {
                dir ('/home/jenkins/go/src/github.com/jenkins-x/jx') {
                    checkout scm
                    container('go') {
                        sh "make"
                        sh "make test"
                        sh "./build/jx --help"
                    }
                }
            }
        }
    
        stage('Build and Release') {
            environment {
                GH_CREDS = credentials('jenkins-x-github')
            }
            when {
                branch 'master'
            }
            steps {
                dir ('/home/jenkins/go/src/github.com/jenkins-x/jx') {
                    checkout scm
                    container('go') {
                        sh "GITHUB_ACCESS_TOKEN=$GH_CREDS_PSW make release"
                    }
                }
            }
        }
    }
}
