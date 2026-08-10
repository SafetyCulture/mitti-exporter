<script>
    export let name = '';
    export let clazz = '';
    export let type = 'neutral'
</script>

<div class="pill {clazz}"
     class:success={type === 'success'}
     class:info={type === 'info'}
     class:neutral={type === 'neutral'}
     class:error={type === 'error'}
     class:cancelled={type === 'cancelled'}
>{name}</div>

<style>
    .pill {
        border-radius: 16px;
        text-align: center;
        width: fit-content;
        block-size: fit-content;
        padding: 5px 15px;
    }

    .success {
        background-color: #E8FDF5;
        color: #005438;
    }

    .info {
        background-color: #E5FAFF;
        color: #0D75B5;
    }

    .neutral {
        background-color: #EEF1F7;
        color: #3F495A;
    }

    .cancelled {
        background-color: #FFF0F1;
        color: #B02010;
    }

    .error {
        background-color: #B02010;
        color: #FFF0F1;
    }
</style>
